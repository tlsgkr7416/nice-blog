#!/bin/bash

aws eks update-kubeconfig --region ap-northeast-2 --name board-eks-cluster
eksctl utils associate-iam-oidc-provider --region ap-northeast-2 --cluster board-eks-cluster --approve

eksctl create iamserviceaccount --region ap-northeast-2 --namespace kube-system --cluster board-eks-cluster --name ebs-csi-controller-sa --role-name ebs-csi-controller-role --attach-policy-arn arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy --approve --role-only

eksctl create addon --name aws-ebs-csi-driver --cluster board-eks-cluster --service-account-role-arn arn:aws:iam::395281289824:role/ebs-csi-controller-role --force

aws iam create-policy --policy-name AWSLoadBalancerControllerIAMPolicy --policy-document file://~/Downloads/blog-new-2/blog/nice-blog/shell/alb-controller-policy.json

eksctl create iamserviceaccount --cluster board-eks-cluster --region ap-northeast-2 --namespace kube-system --name load-balancer-controller --role-name AmazonEKSLoadBalancerControllerRole --attach-policy-arn arn:aws:iam::395281289824:policy/AWSLoadBalancerControllerIAMPolicy --approve

helm repo add eks https://aws.github.io/eks-charts

helm repo update

helm install load-balancer-controller eks/aws-load-balancer-controller -n kube-system --set clusterName=board-eks-cluster --set serviceAccount.create=false --set serviceAccount.name=load-balancer-controller
