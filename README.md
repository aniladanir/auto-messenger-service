# Auto Messenger Service

A background service written in **Golang** that automatically processes and sends pending messages from a PostgreSQL database to a webhook endpoint.


## Installation & Configuration

You can run this project using **Docker Compose** (includes Database + Redis) or **Docker File** (connects to your own Database + Redis).

### Option 1: Quick Start (Docker Compose)
Use this if you want to spin up the App, PostgreSQL, and Redis all at once.

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/yourusername/auto-messenger.git
    cd auto-messenger
    ```

2.  **Run the services:**
    ```bash
    docker compose up --build
    ```
    *The API will be available at `http://localhost:6060`.*

---

### Option 2: Custom Infrastructure
Use this if you already have **PostgreSQL** and **Redis** running (e.g., on AWS RDS, a local machine, or another server) and just want to run the application container.

1.  **Build the Docker Image:**
    ```bash
    docker build -t auto-messenger .
    ```
2. **Set custom configuration**
    You must set your connection details in **config.json** file.
    ```json
    {
        "db_conn_string": "postgres://your-postgres-host.com:5432/messenger",
        "redis_addr": "your-redis-host.com:6379",
        "web_hook_url": "https://your-web-hook",
    }
    ```

3.  **Run the Container:**
    ```bash
    docker run -d -p 6060:6060
    ```
---

### Configuration Reference

Use these variables in config file to determine the behaviour of the application.

| Variable | Description |
| :--- | :--- |
| `http_port` | http server port |
| `db_conn_string` | database connection string |
| `redis_addr` | redis cluster address |
| `web_hook_url` | webhook url |
| `msg_batch_size` | number of messages to be processed in each cycle |
| `msg_send_interval` | interval between each cycle |
| `msg_max_retry` | maximum number of retries for failed messages |

### Preassumptions

Application expects external APIs to return **202 Accepted** status code on success.

If response status code is equal to or above **500 Internal Server Error**, request will be retried.

If response status code is equal to or above **400 Bad Request**, it will be treated as user error and message will be marked as fail.
