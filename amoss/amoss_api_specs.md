![alt text](https://pbs.twimg.com/profile_images/671443083704184832/d0oR1_kX.png)

## Table of Contents
- [1. About the Study](#1-about-the-study)
- [2. AWS Documentation](#2-aws-documentation)
  - [2.1. AWS VPC Dashboard](#21-aws-vpc-dashboard)
    - [VPC](#vpc)
    - [Subnets](#subnets)
    - [Route Tables](#route-tables)
    - [Internet Gateway](#internet-gateway)
    - [Endpoints](#endpoints)
    - [NAT Gateway](#nat-gateway)
    - [Security Groups](#security-groups)
      - [Port Legend](#port-legend)
      - [Load Balancer HA_PROD_LB_SG](#load-balancer-ha_prod_lb_sg)
      - [Jump Box HA_PROD_JB_SG](#jump-box-ha_prod_jb_sg)
      - [Database HA_PROD_RDS_SG](#database-ha_prod_rds_sg)
      - [Vault HA_PROD_VAULT_SG](#vault-ha_prod-vault-sg)
      - [Config Box HA_PROD_CB_SG](#config-box-ha_prod_cb_sg)
      - [API HA_PROD_API_SG](#api-ha_prod_api_sg)
  - [2.2. AWS EC2 Dashboard](#22-aws-ec2-dashboard)
    - [Instances](#instances)
      - [Vault Server](#vault-server)
      - [Config Box](#config-box)
      - [API Server](#api-server)
      - [Jump Box](#jump-box)
    - [Load Balancers](#load-balancers)
      - [Instances](#instances)
      - [Listeners](#listeners)
      - [Health Check](#health-check)
  - [2.3. AWS RDS Dashboard](#23-aws-rds-dashboard)
    - [Engine](#engine)
    - [Database Specifications](#database-specifications)
    - [Encrypt Database](#encrypt-database)
  - [2.4. AWS Server Security Policy](#24-aws-server-security-policy)
  - [2.5. AWS S3](#25-aws-s3)
- [3. Contributors](#3-contributors)


# 1. About the study

* [Emory | Healthy Aging Study](http://healthyaging.emory.edu)
* [Atlanta's biggest ever clinical research study](http://www.bizjournals.com/atlanta/news/2016/03/30/emory-launches-atlantas-biggest-clinical-research.html)
* [Join the Study](https://emory.force.com/ehas/CommunitiesSelfReg)

# 2. AWS Documentation
This is the documentation for the current infrastructure of the Healthy Aging Study's AWS services.

## 2.1. AWS VPC Dashboard

### VPC

- IPV4 CIDR 10.0.0.0/16
- DNS resolution: yes
- Tenancy: Default ***(We set the tenancy to default because in the near future we will apparently be moving to the LITS VPC)***

### Subnets

- Currently there are 4 subnets. A private and public in us-east-1a and us-east-1c.
- The IPV4 CIDR for the subnets are 10.0.1.0/24, 10.0.2.0/24, 10.0.3.0/24, 10.0.4.0/24.

### Route Tables

- We have 1 private route table and 1 public route table.
- The private route table routes all traffic to our NAT gateway.
- The private route table also routes any S3 traffic to a encrypted S3 endpoint provided to us by AWS.
- The private route table is our main route table for the VPC.
- The destination of all traffic on the public route table is to the Internet Gateway(IGW).

### Internet Gateway

- Attach IGW to VPC

### Endpoints

- We have an encrypted tunnel endpoint where all of our S3 traffic goes through.

### Nat Gateway

- NAT Gateway has an elastic IP address and resides in our public subnet.

### Security Groups

#### Port Legend

Protocol | Port
---|---
HTTP | 80
HTTPS | 443

Service | Port
---|---
PostgreSQL | 5432
Vault | 8200

EC2 | Port
---|---
Loadbalancer | 10443
Health Check | 8080

#### Load Balancer HA_PROD_LB_SG
    • Security Group= HA_PROD_LB_SG
    • Inbound= TCP:443
    • Outbound= ALL TRAFFIC
    
#### Jump Box HA_PROD_JB_SG
    • Security Group= HA_PROD_JB_SG
    • Inbound= SSH Dev Computers
    • Outbound= SSH Config Box/ API/ Vault 
    
#### Database HA_PROD_RDS_SG
    • Security Group= HA_PROD_RDS_SG
    • Inbound= TCP:5432 Config Box/ API/ VAULT
    
#### Vault HA_PROD_VAULT_SG
    • Security Group= HA_PROD_VAULT_SG
    • Inbound= SSH Jumpbox/ TCP:8200 API/ 
    • Outbound= TCP:5432 Database
    
#### Config Box HA_PROD_CB_SG
    • Security Group= HA_PROD_CB_SG
    • Inbound= SSH Jumpbox
    • Outbound= TCP:5432 Database (Will evolve later for automation tools and services)
    
#### API HA_PROD_API_SG
    • Security Group= HA_PROD_API_SG
    • Inbound = SSH Jumpbox/ TCP:10443 Loadbalancer/ TCP:8080 Healthcheck 
    • Outbound= ALL TRAFFIC

## 2.2. AWS EC2 Dashboard

### Instances

Right now we have one of each instance in us-east-1a, as there is no need for high availability at this moment. The database is the only instance that has high availability. (Every instance is Ubuntu)

#### Vault Server
    • Resides in private subnet
    • IAM Role: EHAS_VAULT_PROD_ROLE
      i. EC2 Amazon Full Access
    • Enable protect against accidental termination
    • Enable CloudWatch 

#### Config Box
    • Resides in private subnet
    • IAM Role: EHAS_CONFIG_BOX_ROLE
    • Enable protect against accidental termination

#### API Server
    • Resides in private subnet
    • IAM Role: EHAS_PROD_SERVER_ROLE
    • Enable protect against accidental termination

#### Jump Box
    • Resides in private subnet
    • Enable protect against accidental termination

### Load Balancers

#### Instances
    • Availability in us-east-1a and us-east-1c on the public subnet.

#### Listeners
    • The load balancer uses TCP instead of HTTP in order to support client-side certification functionality on the instance level.
    • Port 443 is mapped to 10443 on the API instance.
  
#### Health Check
    • Pings port 8080 on instances.

## 2.3. AWS RDS Dashboard

### Engine
    • Postgres database engine.
    
### Database Specifications
    • DB Instance Class: t2 small 1vCPU 2gib RAM
    • Multi-AZ Deployment: Enabled
    • Allocated Storage: 100GB

### Encrypt Database

There was an option to encrypt the database but it would need a more expensive DB instance class, so we opted to go without it since this instance is in our private subnet and not available to the internet.

## 2.4. AWS Server Security Policy

Updates to the servers will be made every Monday morning. We will briefly allow outbound traffic in our security groups from the servers and apt-get update/upgrade for our ubuntu servers.

Updates to the application dependencies will be updated every first Monday of each month with the latest compatible versions. We will briefly allow outbound traffic in our security groups so that updates will be possible.

## 2.5. AWS S3

Currently holds CSV files corresponding to certain games 
Bucket: emory-healthy-aging-data
Folders: dev, qa, prod
    
# 3. Contributors

Christoper Wright Aaron && Tony Nguyen

**README** documentation by **_Tony Nguyen_**
