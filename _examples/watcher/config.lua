config = {
    features = {
        enable_cache = true,
        cache_ttl = 300,
        sampling_rate = 0.1,
        enable_metrics = true
    },
    limits = {
        max_connections = 1000,
        request_timeout = "30s",
        rate_limit = 100
    }
} 