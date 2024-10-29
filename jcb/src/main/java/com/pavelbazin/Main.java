package com.pavelbazin;


import com.pavelbazin.circuitbreaker.CircuitBreaker;

import java.time.Duration;
import java.util.function.Function;

public class Main {
    /**
     * Simulate remote service behavior.
     *
     * Useful service behaviors are:
     *  - Always failing `createService(600, 2500, 1)`
     *  - Never failing `createService(600, 2500, 0.00)`
     *
     * @param latencyFrom service latency lower bound
     * @param latencyTo service latency upper bound
     * @param failureRate service failure rate %, so that 0.65% = 65% failure chance
     * @return pre configured service
     */
    public static Function<Integer, String> createService(int latencyFrom, int latencyTo, double failureRate) {
        return in -> {
            int latency = latencyFrom + (int) (Math.random() * (latencyTo - latencyFrom));

            try {
                Thread.sleep(latency);
                // Simulate service failure
                if (Math.random() < failureRate) {
                    System.out.println("service: fail");
                    throw new RuntimeException("service failure");
                }

                System.out.println("service: ok");
                return "result is " + in;
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                throw new RuntimeException(e);
            }
        };
    }

    public static void main(String[] args) throws InterruptedException {
        var cb = new CircuitBreaker(
                2,
                2,
                Duration.ofSeconds(2),
                Duration.ofSeconds(5));

        Function<Integer, String> service = createService(600, 2500, 0.60);

        for (var i = 0; i < 32; i++) {
            var res = cb.call(i, service);
            if (res.isEmpty()) {
                // Wait a sec here before making a new call
                System.out.println("waiting after failure");
                Thread.sleep(1000);
            }

            System.out.printf("processed %d calls%n", i);
        }
    }
}