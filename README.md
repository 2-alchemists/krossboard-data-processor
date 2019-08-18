# Standalone Multi-cluster Instance of Kubernetes Opex Analytics
 
Goals:

  * Handle Kubernetes clusters automatically discovered from `.kubeconfig`, in a native way than kubectl does.
  * Provide Marketplace based automatic deployment on public clouds (AWS, GCP, Azure). First targets: AWS and GCP. 
  * Include utility tools to help automatically provisioing `.kubeconfig` on public clouds (e.g. using `$ gcloud container clusters get-credentials` on GCP). 
