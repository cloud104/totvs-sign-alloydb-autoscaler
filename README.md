[![AlloyDB Logo](https://miro.medium.com/v2/resize:fit:1400/1*X8fQjt3rQwXHc0lpnW7MXA.png)](#---)


# AlloyDB Autoscaler

---
======================

## Description

------------

The AlloyDB Autoscaler is an application that automatically scales the number of read replicas of an AlloyDB cluster based on CPU and memory usage.

## Environment Variables

-------------------------

The following environment variables are essential for the application to function:

* `GOOGLE_APPLICATION_CREDENTIALS`: path to the Google Cloud credentials file
* `GCP_PROJECT`: Google Cloud project ID
* `CLUSTER_NAME`: AlloyDB cluster name
* `INSTANCE_NAME`: AlloyDB read instance name
* `REGION`: region where the AlloyDB cluster is located
* `LOG_LEVEL`: log level for the application
* `CPU_THRESHOLD`: CPU usage threshold for scaling (in percentage)
* `MEMORY_THRESHOLD`: memory usage threshold for scaling (in percentage)
* `CHECK_INTERVAL`: time interval between checks (in seconds)
* `EVALUATION`: time window to evaluate checks before scaling up or down (in seconds)
* `MIN_REPLICAS`: minimum number of replicas allowed
* `MAX_REPLICAS`: maximum number of replicas allowed
* `TIMEOUT_SECONDS`: GCP API timeout (in seconds)


## Requirements

--------------

* Configured Google Cloud credentials file
* Access to an existing AlloyDB cluster in Google Cloud
* Configuration of environment variables (via `.env` file for local execution or via Kubernetes deployment for cloud environments)

Since the application is containerized in Docker, it is not necessary to have Go installed locally or the Google Cloud SDK, as these components will already be included in the Docker image.

## How It Works

----------------

The application works as follows:

1. Reads the environment variables and configures the Google Cloud client
2. Checks CPU and MEMORY usage in GCP Cloud Monitoring of the AlloyDB cluster at each specified time interval
3. Evaluates all checks in a time window before making decisions
4. If CPU or MEMORY usage exceeds the specified threshold, scales up the number of cluster replicas by +1
5. If CPU and MEMORY usage is below the threshold and there is more than one replica, reduces the number of replicas by -1 until it reaches the desired minimum value.

### Using with Docker

To use the application with Docker, follow the steps below:

1. Fill in the `.env` file with the necessary environment variables for local execution.
2. Run the command `docker build -t alloydb-autoscaler .` to build the Docker image
3. Run the command `docker run -v ./.env:/app/.env -v ./key.json:/app/key.json alloydb-autoscaler && docker logs -f -t alloydb-autoscaler` to run the application and display the logs.

**IMPORTANT**: The rule for scaling down only applies if the nodepoolcount is greater than 1. If there is only 1 node (the minimum), the program will only consider the possibility of scaling up