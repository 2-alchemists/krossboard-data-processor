![](krossboard-architecture-thumbnail.png)


![Test and build](https://github.com/2-alchemists/krossboard-data-processor/workflows/test-and-build/badge.svg)
![golangci-lint](https://github.com/2-alchemists/krossboard-data-processor/workflows/golangci-lint/badge.svg)
![Build AWS images](https://github.com/2-alchemists/krossboard-data-processor/workflows/build-aws-images/badge.svg)
![Build GCP images](https://github.com/2-alchemists/krossboard-data-processor/workflows/build-gcp-images/badge.svg)
![Build Azure images](https://github.com/2-alchemists/krossboard-data-processor/workflows/build-azure-images/badge.svg)

---

# Overview
`krossboard-data-processor` is the backend component of [Krossboard](https://github.com/2-alchemists/krossboard).

Key features of Krossboard:

 * **Multi-Kubernetes Data Collection**: Periodically collecting raw metrics related to containers, pods and nodes from several Kubernetes source clusters. By design, the data collection period is 5 minutes.
 * **Powerful Analytics Processing**: Internally process raw metrics to produce insightful Kubernetes usage accounting and analytics metrics. By design, these analytics metrics are handled on a hourly-basis, for every namespace, cluster, and also globally.
 * **Insightful Usage Accounting**: Periodically process usage accounting, for each cluster and cluster's namespace. In the current implementation, the accounting is handled by design for the following period:  daily for the 14 last days, monthly for the 12 last months.
 * **REST API**: Exposes the generated analytics data to third-party systems. For example, [Krossboard UI](https://github.com/2-alchemists/krossboard-ui) consumes this API to produce its charts and dashboards. 
 * **Easy to deploy**: Krossboard can be easily deployed through a virtual machine on-premises or on public clouds. In a close future, it's expected to be deployed through a Kubernetes operator. It can be set up to automatically discover and handle Kubernetes clusters backed by Amazon EKS, Google GKE, Microsoft AKS.  

![](krossboard-architecture-overview.png)


# Getting Started

* [Setup Krossboard for Multi-Cloud or Cross-Kubernetes Distributions](https://krossboard.app/docs/60_deploy-for-cross-cloud-and-on-premises-kubernetes/)
* [Setup Krossboard for Amazon EKS](https://krossboard.app/docs/50_deploy-for-amazon-eks/)
* [Setup Krossboard for Azure AKS](https://krossboard.app/docs/30_deploy-for-azure-aks/)
* [Setup Krossboard for Google GKE](https://krossboard.app/docs/20_deploy-for-google-gke/)

# Open to Contributions
We encourage feedback and always make our best to handle any issues as fast as possible. 

Don't hesitate to submit a issue or make a feature request.

All contributions, including, code and documentation, are accepted under the terms of [GNU Affero General Public License](LICENSE.md).

Read the [code of conduct](CODE-OF-CONDUCT.md) to learn more about our contribution guideline. 

[![StackShare](http://img.shields.io/badge/tech-stack-0690fa.svg?style=flat)](https://stackshare.io/2alchemists/krossboard)
