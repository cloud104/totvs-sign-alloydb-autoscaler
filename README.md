[![AlloyDB Logo](https://miro.medium.com/v2/resize:fit:1400/1*X8fQjt3rQwXHc0lpnW7MXA.png)](#)


# AlloyDB Autoscaler

## Description

The AlloyDB Autoscaler is an application that automatically scales the number of read replicas of an AlloyDB cluster based on CPU and memory usage metrics.

## Environment Variables

The following environment variables are essential for the application to function:

* `GOOGLE_APPLICATION_CREDENTIALS`: Path to the Google Cloud credentials file
* `GCP_PROJECT`: Google Cloud project ID
* `CLUSTER_NAME`: AlloyDB cluster name
* `INSTANCE_NAME`: AlloyDB read instance name
* `REGION`: Region where the AlloyDB cluster is located
* `LOG_LEVEL`: Log level for the application (debug, info, warn, error)
* `CPU_THRESHOLD`: CPU usage threshold for scaling (in percentage)
* `MEMORY_THRESHOLD`: Memory usage threshold for scaling (in percentage)
* `CHECK_INTERVAL`: Time interval between checks (in seconds)
* `EVALUATION`: Time window to evaluate checks before scaling up or down (in seconds)
* `MIN_REPLICAS`: Minimum number of replicas allowed
* `MAX_REPLICAS`: Maximum number of replicas allowed
* `TIMEOUT_SECONDS`: GCP API timeout (in seconds)

### Example .env File

```
GOOGLE_APPLICATION_CREDENTIALS=/app/key.json
GCP_PROJECT=my-gcp-project
CLUSTER_NAME=my-alloydb-cluster
INSTANCE_NAME=my-read-instance
REGION=us-central1
LOG_LEVEL=info
CPU_THRESHOLD=70
MEMORY_THRESHOLD=70
CHECK_INTERVAL=60
EVALUATION=300
MIN_REPLICAS=1
MAX_REPLICAS=5
TIMEOUT_SECONDS=120
```

## Requirements

* Configured Google Cloud credentials file (key.json)
* Access to an existing AlloyDB cluster in Google Cloud
* Configuration of environment variables (via `.env` file for local execution or via Kubernetes deployment for cloud environments)

Since the application is containerized in Docker, it is not necessary to have Go installed locally or the Google Cloud SDK, as these components are included in the Docker image.

## Service Account Setup

1. Create a service account in the Google Cloud Console with the required permissions (listed below)
2. Generate and download the service account key file (JSON format)
3. Rename the downloaded file to `key.json` and place it in the root directory of the project
4. Make sure the path in `GOOGLE_APPLICATION_CREDENTIALS` environment variable points to this file

## How It Works

The application follows this workflow:

1. Reads the environment variables and configures the Google Cloud client
2. Checks CPU and memory usage in GCP Cloud Monitoring of the AlloyDB cluster at each specified time interval
3. Evaluates all checks in a time window before making scaling decisions
4. If CPU or memory usage exceeds the specified threshold, scales up the number of cluster replicas by 1
5. If CPU and memory usage is below the threshold and there is more than one replica, reduces the number of replicas by 1 until it reaches the minimum value

**IMPORTANT**: The rule for scaling down only applies if the current replica count is greater than the minimum replicas setting. If there is only the minimum number of replicas, the application will only consider the possibility of scaling up.

## Deployment

### Using with Docker

To use the application with Docker:

1. Fill in the `.env` file with the necessary environment variables
2. Place your service account key file as `key.json` in the project root directory
3. Build the Docker image:
   ```
   docker build -t alloydb-autoscaler .
   ```
4. Run the container:
   ```
   docker run -d --name alloydb-autoscaler \
     -v $(pwd)/.env:/app/.env \
     -v $(pwd)/key.json:/app/key.json \
     alloydb-autoscaler:latest
   ```

### Using with Kubernetes

To deploy the application on Kubernetes:

1. Create a ConfigMap for the environment variables:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alloydb-autoscaler-config
data:
  GCP_PROJECT: "my-gcp-project"
  CLUSTER_NAME: "my-alloydb-cluster"
  INSTANCE_NAME: "my-read-instance"
  REGION: "us-central1"
  LOG_LEVEL: "info"
  CPU_THRESHOLD: "70"
  MEMORY_THRESHOLD: "70"
  CHECK_INTERVAL: "60"
  EVALUATION: "300"
  MIN_REPLICAS: "1"
  MAX_REPLICAS: "5"
  TIMEOUT_SECONDS: "120"
```

2. Create a Secret for the service account key:

```bash
kubectl create secret generic alloydb-sa-key --from-file=key.json=./key.json
```

3. Create a Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alloydb-autoscaler
  labels:
    app: alloydb-autoscaler
spec:
  replicas: 1
  selector:
    matchLabels:
      app: alloydb-autoscaler
  template:
    metadata:
      labels:
        app: alloydb-autoscaler
    spec:
      containers:
      - name: alloydb-autoscaler
        image: [YOUR_REGISTRY]/alloydb-autoscaler:latest
        envFrom:
        - configMapRef:
            name: alloydb-autoscaler-config
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: "/var/secrets/google/key.json"
        volumeMounts:
        - name: google-cloud-key
          mountPath: /var/secrets/google
      volumes:
      - name: google-cloud-key
        secret:
          secretName: alloydb-sa-key
```

4. Apply the configuration:

```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

## Required Permissions for Service Account

The Google Cloud service account used by the application needs the following permissions:

- `alloydb.clusters.get`
- `alloydb.clusters.list`
- `alloydb.instances.get`
- `alloydb.instances.list`
- `alloydb.instances.update`
- `alloydb.locations.get`
- `alloydb.locations.list`
- `alloydb.operations.get`
- `alloydb.operations.list`
- `alloydb.users.get`
- `alloydb.users.list`
- `monitoring.timeSeries.list`

Alternatively, you can assign the following predefined roles:
- `roles/alloydb.admin`
- `roles/monitoring.viewer`

## Troubleshooting

### Logs

For more detailed logs, set `LOG_LEVEL=debug` in the `.env` file or ConfigMap.

To check logs in Kubernetes:
```bash
kubectl logs -f deployment/alloydb-autoscaler
```

### Connectivity Issues

Ensure that the container has internet access to communicate with Google Cloud APIs.

For Kubernetes deployments, verify that the cluster has outbound internet access or appropriate firewall rules.

### Permission Issues

If you encounter permission errors, verify that the service account has all the necessary permissions listed above.

### Common Errors

- **"Cannot find credentials"**: Ensure the key.json file is correctly mounted and the GOOGLE_APPLICATION_CREDENTIALS path is correct.
- **"Permission denied"**: Verify that the service account has all required permissions.
- **"Cluster not found"**: Check that the CLUSTER_NAME and REGION values are correct.

## Contribution

Contributions are welcome! Feel free to open issues or send pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.