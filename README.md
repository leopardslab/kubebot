### Hi There, This project is a part of GSoC 2022 under SCoReLab.

## Project Statement:

Kubernetes is a great tool for container orchestration, running Kubernetes in your production environment is getting traction in the Cloud industry. With growing DevOps tools, it is now a tedious and time-consuming task SREs and developers to continuously monitor their remote applications running inside a multicluster Kubernetes environment. Though there are some great tools using which we can monitor Kubernetes tools there is a need for easy deployment setup and an alerting tool that people can use to get alerts when something goes wrong inside their application running in a Kubernetes environment. It is often time-consuming to figure out the root cause of such issues, and most importantly not all alerts and issues are important and do not need human interaction.

KubeBot is a smart tool that pulls out of box metrics, traces, events and logs collection for applications running inside Kubernetes and reports all the collected data on a single dashboard and push notifications to users for critical ones. The collected metrics, traces, events and logs will help to debug the application running inside Kubernetes faster and effectively. A main feature of KubeBot is that after collecting traces, it does a root-cause analysis and sends alerts to the user regarding fixing the issue occuring in the cluster.

## Project High Level Design:

The project's workflow is as shown in the figure below, as we move ahead towards the execution, it may subject to change.

[Project Link on GSoC Portal](https://summerofcode.withgoogle.com/programs/2022/projects/CrF9bzaI)


![KubeBot Design](https://user-images.githubusercontent.com/48913548/174438534-310ac132-64fb-4e7e-9c74-f21fd864a508.jpeg)

