package com.pavelbazin.circuitbreaker;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.function.Function;

import static com.pavelbazin.Main.createService;


class CircuitBreakerTest {
    private CircuitBreaker cb;

    @BeforeEach
    void setUp() {
        cb = new CircuitBreaker(
                2,
                2,
                Duration.ofSeconds(2),
                Duration.ofSeconds(5)
        );
    }

    @Test
    void testCircuitBreakerCall_neverFailing() {
        // Never failing service
        Function<Integer, String> service = createService(600, 2500, 0.00);
        var res = cb.call(1, service);

        assert res.isPresent() : "res should not be null";
        assert res.get().equals("result is 1") : "res should be equal to 1";
    }

        @Test
    void testCircuitBreakerCall_alwaysFailing() {
        // Never failing service
        Function<Integer, String> service = createService(600, 2500, 1);
        var res = cb.call(1, service);

        assert res.isEmpty() : "res should be null";
    }
}