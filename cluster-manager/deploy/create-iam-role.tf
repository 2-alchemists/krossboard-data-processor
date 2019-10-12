resource "aws_iam_role" "role_koamc_cluster_manager" {
  name               = "koamc-cluster-manager"
  description = "Role to attach permissions to KOAMC Cluster Manager EC2 instance"
  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "sts:AssumeRole",
            "Principal": {
                "Service": "ec2.amazonaws.com"
            }
        }
    ]
}
EOF
}

resource "aws_iam_policy" "policy_koamc_cluster_manager" {
  name        = "koamc-cluster-manager-policy"
  description = "Attach role and policies for KOAMC cluster manager"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "eks:ListClusters",
                "eks:DescribeCluster"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "koamc_cluster_manager_rolepolicy_attachment" {
  role       = "${aws_iam_role.role_koamc_cluster_manager.name}"
  policy_arn = "${aws_iam_policy.policy_koamc_cluster_manager.arn}"
}