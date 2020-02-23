#!/bin/bash
export AWS_DEFAULT_REGION="eu-central-1"
KROSSBOARD_AMI='ami-0d1067b6dd89a5731'
KROSSBOARD_SG='sg-0223dc619eef80444'
KROSSBOARD_ROLE='krossboard-role-test'
KROSSBOARD_INSTANCE_PROFILE="krossboard-instance-profile"
CLUSTER_VERSION="1.14"
CLUSTER_NAME="eks-krossboard-demo-1"
DEFAULT_NODEGROUP="${CLUSTER_NAME}-default-pool"
SSH_PUB_KEY='krossboard-test'
DESIRED_TYPE='t2.small'
MIN_NODES=1
MAX_NODES=1
DESIRED_NODES=1
THIS_DIR=$PWD

eksctl create cluster \
  --name "${CLUSTER_NAME}" \
  --region="${AWS_DEFAULT_REGION}" \
  --version="${CLUSTER_VERSION}" \
  --nodegroup-name="${DEFAULT_NODEGROUP}" \
  --node-type=${DESIRED_TYPE} \
  --nodes=${DESIRED_NODES} \
  --nodes-min=${MIN_NODES} \
  --nodes-max=${MAX_NODES} \
  --node-private-networking \
  --ssh-public-key="${SSH_PUB_KEY}"


# install metrics server
cd ~/Downloads
DOWNLOAD_URL=$(curl -Ls "https://api.github.com/repos/kubernetes-sigs/metrics-server/releases/latest" | jq -r .tarball_url)
DOWNLOAD_VERSION=$(grep -o '[^/v]*$' <<< $DOWNLOAD_URL)
curl -Ls $DOWNLOAD_URL -o metrics-server-$DOWNLOAD_VERSION.tar.gz
mkdir metrics-server-$DOWNLOAD_VERSION
tar -xzf metrics-server-$DOWNLOAD_VERSION.tar.gz --directory metrics-server-$DOWNLOAD_VERSION --strip-components 1
kubectl apply -f metrics-server-$DOWNLOAD_VERSION/deploy/1.8+/
cd -

# create clusterrole and binding for Krossboard
kubectl create -f $THIS_DIR/deploy/k8s/clusterrolebinding-eks.yml

KROSSBOARD_INSTANCE=$(aws ec2 run-instances \
    --image-id ${KROSSBOARD_AMI} \
    --count 1 \
    --instance-type t2.micro \
    --key-name "${SSH_PUB_KEY}" \
    --security-group-ids ${KROSSBOARD_SG} | jq -r .Instances[0].InstanceId -)

aws iam create-instance-profile \
    --instance-profile-name ${KROSSBOARD_INSTANCE_PROFILE}

aws iam add-role-to-instance-profile \
    --instance-profile-name ${KROSSBOARD_INSTANCE_PROFILE} \
    --role-name ${KROSSBOARD_ROLE}

aws ec2 associate-iam-instance-profile \
    --iam-instance-profile Name=${KROSSBOARD_INSTANCE_PROFILE} \
    --instance-id ${KROSSBOARD_INSTANCE}

