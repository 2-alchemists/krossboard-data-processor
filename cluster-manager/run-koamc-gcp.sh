export KOAMC_LOG_LEVEL=debug
export KOAMC_GCP_METADATA_SERVICE=http://127.0.0.1:8000
export GOOGLE_APPLICATION_CREDENTIALS=/home/ubuntu/.gcp/serviceaccount/gcp_credentials_koamc_cluster_manager.json
export KOAMC_KOAINSTANCE_IMAGE='rchakode/kube-opex-analytics:0.4.8'
make run
