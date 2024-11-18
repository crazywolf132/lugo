service = {
    name = "advanced-service",
    version = "2.1.0",
    environment = "prod",

    network = {
        host = "api.example.com",
        port = 8443,
        read_timeout = "30s",
        write_timeout = "30s",
        tls = {
            enabled = true,
            cert_file = "/etc/certs/server.crt",
            key_file = "/etc/certs/server.key",
            min_version = "1.3",
            ciphers = {
                "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
                "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
            }
        }
    },

    databases = {
        main = {
            driver = "postgres",
            host = "db.example.com",
            port = 5432,
            name = "maindb",
            username = "app_user",
            password = "{{ env "DB_PASSWORD" }}",
            parameters = {
                sslmode = "verify-full",
                pool_size = "20"
            },
            max_connections = 100,
            idle_timeout = "5m"
        },
        analytics = {
            driver = "mysql",
            host = "analytics.example.com",
            port = 3306,
            name = "analytics",
            username = "analyst",
            password = "{{ env "ANALYTICS_DB_PASSWORD" }}",
            max_connections = 50,
            idle_timeout = "10m"
        }
    },

    cache = {
        type = "redis",
        servers = {
            "cache-1.example.com:6379",
            "cache-2.example.com:6379",
            "cache-3.example.com:6379"
        },
        key_prefix = "adv:",
        ttl = "1h"
    },

    features = {
        new_ui = {
            enabled = true,
            percentage = 75.5,
            allowed_ips = {
                "10.0.0.0/8",
                "172.16.0.0/12"
            },
            allowed_users = {
                "beta@example.com",
                "test@example.com"
            },
            start_time = "2024-01-01T00:00:00Z",
            end_time = "2024-12-31T23:59:59Z"
        },
        experimental_api = {
            enabled = false,
            percentage = 10.0,
            allowed_ips = {
                "192.168.1.0/24"
            }
        }
    },

    monitoring = {
        enabled = true,
        metrics = {
            port = 9090,
            path = "/metrics",
            interval = "15s"
        },
        tracing = {
            enabled = true,
            endpoint = "http://jaeger:14268/api/traces",
            sample_rate = 0.1,
            batch_size = 100
        },
        health_check = {
            path = "/health",
            interval = "30s",
            timeout = "5s"
        }
    },

    limits = {
        max_requests = 10000,
        max_concurrent = 1000,
        rate_limit = 100.0,
        burst_limit = 200,
        request_timeout = "30s",
        shutdown_timeout = "60s",
        max_request_size = 5242880  -- 5MB
    }
} 