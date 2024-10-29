package com.pavelbazin.circuitbreaker;

import java.lang.ref.Cleaner;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.concurrent.*;
import java.util.function.Function;

public class CircuitBreaker {
    private enum State {
        CLOSED,
        OPEN,
        HALF_OPEN,
    }

    // Variable state fields
    private State state = State.CLOSED;
    private int failureCount = 0;
    private int successCount = 0;
    private Instant lastFailureTime = Instant.EPOCH;

    // User defined fields
    private final int failureThreshold;
    private final int halfOpenThreshold;
    private final Duration recoveryTime;
    private final Duration timeout;

    // Executor and clean up logic
    private final ExecutorService executor = Executors.newFixedThreadPool(8);
    private static final Cleaner cleaner = Cleaner.create();

    private record CleanUp(ExecutorService executor) implements Runnable {
        @Override
        public void run() {
            executor.shutdown();
        }
    }

    /**
     * Circuit breaker.
     * @param failureThreshold defines the fail tolerance which is how many calls may
     *                         fail before circuit breaker starts to block requests.
     * @param halfOpenThreshold defines the recovery threshold which is how many calls
     *                          needs to succeed before circuit breaker will allow all
     *                          the traffic again.
     * @param recoveryTime defines time needs to pass since the last failure before
     *                     circuit breaker will attempt to perform a call again.
     * @param timeout defines response timeout.
     */
    public CircuitBreaker(int failureThreshold, int halfOpenThreshold, Duration recoveryTime, Duration timeout) {
        this.failureThreshold = failureThreshold;
        this.halfOpenThreshold = halfOpenThreshold;
        this.recoveryTime = recoveryTime;
        this.timeout = timeout;

        // Clean up resources upon GC
        cleaner.register(this, new CleanUp(executor));
    }

    private void log(String message) {
        System.out.println("CircuitBreaker: " + message);
    }

    /**
     * Perform a service call through a circuit breaker.
     *
     * @param input service input value
     * @param fn service function to be called within circuit breaker
     * @return result of the service call
     * @param <T> input type parameter
     * @param <R> service return type parameter
     */
    synchronized public <T, R> Optional<R> call(T input, Function<T, R> fn) {
        log("processing a new call");

        return switch (this.state) {
            case CLOSED -> {
                log("switch -- processClosedState");
                yield processClosedState(input, fn);
            }
            case OPEN -> {
                log("switch -- processOpenState");
                yield processOpenState();
            }
            case HALF_OPEN -> {
                log("switch -- processHalfOpenState");
                yield processHalfOpenState(input, fn);
            }
        };
    }

    private void resetCircuit() {
        state = State.CLOSED;
        failureCount = 0;
        successCount = 0;
        lastFailureTime = Instant.EPOCH;
    }

    private <T, R> Optional<R> callWithTimeout(T input, Function<T, R> fn) throws TimeoutException, ExecutionException, InterruptedException {
        Future<Optional<R>> future = executor.submit(() -> Optional.of(fn.apply(input)));

        try {
            return future.get(timeout.toSeconds(), TimeUnit.SECONDS);
        } catch (TimeoutException e) {
            log(String.format("callWithTimeout: failed with %s", e));
            future.cancel(true);
            throw e;
        }
    }

    private <T, R> Optional<R> processClosedState(T input, Function<T, R> fn) {
        log("processing in closed state");

        try {
            return callWithTimeout(input, fn);
        } catch (Exception e) {
            failureCount++;
            lastFailureTime = Instant.now();

            log(String.format("request failed; failureCount = %s;", failureCount));

            if (failureCount >= failureThreshold) {
                log("state transitioning to `open`");
                state = State.OPEN;
            }

            return Optional.empty();
        }
    }

    private <T> Optional<T> processOpenState() {
        log("processing in open state");

        var diff = Duration.between(lastFailureTime, Instant.now()).getSeconds();
        if (diff > recoveryTime.getSeconds()) {
            log("state transitioning to `half-open`");
            state = State.HALF_OPEN;
            failureCount = 0;
        }
        return Optional.empty();
    }

    private <T, R> Optional<R> processHalfOpenState(T input, Function<T, R> fn) {
        log("processing in half-open state");

        Optional<R> res;

        try {
            res = callWithTimeout(input, fn);
        } catch (Exception e) {
            log("processHalfOpenState failed");
            log("state transitioning to `open`");
            failureCount++;
            state = State.OPEN;
            lastFailureTime = Instant.now();

            return Optional.empty();
        }

        // Operation was successful
        successCount++;

        if (successCount >= halfOpenThreshold) {
            log("state transitioning to `closed`");
            resetCircuit();
        }

        return res;
    }
}