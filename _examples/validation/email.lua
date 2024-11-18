email = {
    smtp = {
        host = "smtp.example.com",
        port = 587,
        username = "notifications@example.com",
        password = "secure_password_123",
        tls = true
    },
    defaults = {
        from = "no-reply@example.com",
        reply_to = "support@example.com",
        timeout = "30s",
        retries = 3,
        max_size = 5242880  -- 5MB
    },
    templates = {
        {
            name = "welcome",
            subject = "Welcome to Our Service",
            cc = {"sales@example.com"},
            bcc = {"analytics@example.com"}
        },
        {
            name = "password_reset",
            subject = "Password Reset Request",
            cc = {"security@example.com"}
        }
    }
} 