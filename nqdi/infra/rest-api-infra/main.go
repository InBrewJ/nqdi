package main

import (
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"

	"log"

	crdb_cluster "cdk.tf/go/stack/generated/cockroachdb/cockroach/cluster"
	crdb "cdk.tf/go/stack/generated/cockroachdb/cockroach/provider"
	alb "github.com/cdktf/cdktf-provider-aws-go/aws/v19/alb"
	alblistener "github.com/cdktf/cdktf-provider-aws-go/aws/v19/alblistener"
	albtargetgroup "github.com/cdktf/cdktf-provider-aws-go/aws/v19/albtargetgroup"
	awsecs "github.com/cdktf/cdktf-provider-aws-go/aws/v19/ecscluster"
	ecsservice "github.com/cdktf/cdktf-provider-aws-go/aws/v19/ecsservice"
	ecstaskdefinition "github.com/cdktf/cdktf-provider-aws-go/aws/v19/ecstaskdefinition"
	eip "github.com/cdktf/cdktf-provider-aws-go/aws/v19/eip"
	iampolicyattachment "github.com/cdktf/cdktf-provider-aws-go/aws/v19/iampolicyattachment"
	iamrole "github.com/cdktf/cdktf-provider-aws-go/aws/v19/iamrole"
	"github.com/cdktf/cdktf-provider-aws-go/aws/v19/instance"
	awsec2 "github.com/cdktf/cdktf-provider-aws-go/aws/v19/internetgateway"
	natgateway "github.com/cdktf/cdktf-provider-aws-go/aws/v19/natgateway"
	awsprovider "github.com/cdktf/cdktf-provider-aws-go/aws/v19/provider"
	route "github.com/cdktf/cdktf-provider-aws-go/aws/v19/route"
	routetable "github.com/cdktf/cdktf-provider-aws-go/aws/v19/routetable"
	routetableassociation "github.com/cdktf/cdktf-provider-aws-go/aws/v19/routetableassociation"
	subnet "github.com/cdktf/cdktf-provider-aws-go/aws/v19/subnet"
	awsvpc "github.com/cdktf/cdktf-provider-aws-go/aws/v19/vpc"
	"github.com/joho/godotenv"
)

func NewMyStack(scope constructs.Construct, id string) cdktf.TerraformStack {

	env, err := godotenv.Read()

	if err != nil {
		log.Fatal("REST-API-INFRA ERROR: cannot load .env", err)
	}

	stack := cdktf.NewTerraformStack(scope, &id)

	awsprovider.NewAwsProvider(stack, jsii.String("AWS"), &awsprovider.AwsProviderConfig{
		Region:    jsii.String(env["AWS_REGION"]),
		AccessKey: jsii.String(env["AWS_ACCESS_KEY_ID"]),
		SecretKey: jsii.String(env["AWS_SECRET_ACCESS_KEY"]),
	})

	instance := instance.NewInstance(stack, jsii.String("compute"), &instance.InstanceConfig{
		Ami:          jsii.String("ami-0a628e1e89aaedf80"),
		InstanceType: jsii.String("t2.micro"),
	})

	cdktf.NewTerraformOutput(stack, jsii.String("public_ip"), &cdktf.TerraformOutputConfig{
		Value: instance.PublicIp(),
	})

	return stack
}

func CockroachDbTest(scope constructs.Construct, id string) cdktf.TerraformStack {
	// provider added and go code generated with:
	// cdktf provider add cockroachdb/cockroach
	// correct import path for generated code living locally:
	// "cdk.tf/go/stack/generated/cockroachdb/cockroach/provider"

	env, err := godotenv.Read()

	if err != nil {
		log.Fatal("REST-API-INFRA ERROR: cannot load .env", err)
	}

	stack := cdktf.NewTerraformStack(scope, &id)

	crdb.NewCockroachProvider(stack, jsii.String("Cockroach-test"), &crdb.CockroachProviderConfig{
		Apikey: jsii.String(env["CRDB_API_KEY"]),
	})

	crdb_cluster.NewCluster(stack, jsii.String("crdb_cluster"), &crdb_cluster.ClusterConfig{
		Name:             jsii.String("nqdi-delete-me"),
		CloudProvider:    jsii.String("GCP"),
		Plan:             jsii.String("BASIC"),
		DeleteProtection: jsii.Bool(false),
		Regions:          &[]*crdb_cluster.ClusterRegions{{Name: jsii.String("us-east1")}},
		Serverless: &crdb_cluster.ClusterServerless{
			SpendLimit: jsii.Number(20.0),
		},
	})

	return stack
}

func RestApiInfraFargate(scope constructs.Construct, id string) cdktf.TerraformStack {

	/*

		See nqdi/infra/rest-api-infra/last_cdktf_error.log for clues towards a fix or two

		Remember the docs!
		https://github.com/cdktf/cdktf-provider-aws/blob/main/docs/API.go.md

	*/

	env, err := godotenv.Read()

	if err != nil {
		log.Fatal("REST-API-INFRA ERROR: cannot load .env", err)
	}

	stack := cdktf.NewTerraformStack(scope, &id)

	awsprovider.NewAwsProvider(stack, jsii.String("AWS"), &awsprovider.AwsProviderConfig{
		Region:    jsii.String(env["AWS_REGION"]),
		AccessKey: jsii.String(env["AWS_ACCESS_KEY_ID"]),
		SecretKey: jsii.String(env["AWS_SECRET_ACCESS_KEY"]),
	})

	// (Optional) NAT gateway for outbound internet access from private subnets
	vpc := awsvpc.NewVpc(stack, jsii.String("nqdi-vpc"), &awsvpc.VpcConfig{
		CidrBlock: jsii.String("10.0.0.0/16"),
	})

	igw := awsec2.NewInternetGateway(stack, jsii.String("nqdi-igw"), &awsec2.InternetGatewayConfig{
		VpcId: vpc.Id(),
	})

	igw.Count()

	publicSubnet := subnet.NewSubnet(stack, jsii.String("nqdi-public-subnet"), &subnet.SubnetConfig{
		VpcId:            vpc.Id(),
		CidrBlock:        jsii.String("10.0.1.0/24"),
		AvailabilityZone: jsii.String(env["AWS_REGION"] + "a"),
	})

	eip := eip.NewEip(stack, jsii.String("nqdi-nat-eip"), &eip.EipConfig{
		Vpc: jsii.Bool(true),
	})

	natGateway := natgateway.NewNatGateway(stack, jsii.String("nqdi-nat-gateway"), &natgateway.NatGatewayConfig{
		AllocationId: eip.Id(),
		SubnetId:     publicSubnet.Id(),
	})

	privateSubnet := subnet.NewSubnet(stack, jsii.String("nqdi-private-subnet"), &subnet.SubnetConfig{
		VpcId:            vpc.Id(),
		CidrBlock:        jsii.String("10.0.2.0/24"),
		AvailabilityZone: jsii.String(env["AWS_REGION"] + "a"),
	})

	routeTable := routetable.NewRouteTable(stack, jsii.String("nqdi-private-route-table"), &routetable.RouteTableConfig{
		VpcId: vpc.Id(),
	})

	route.NewRoute(stack, jsii.String("nqdi-private-route"), &route.RouteConfig{
		RouteTableId:         routeTable.Id(),
		DestinationCidrBlock: jsii.String("0.0.0.0/0"),
		NatGatewayId:         natGateway.Id(),
	})

	routetableassociation.NewRouteTableAssociation(stack, jsii.String("nqdi-private-route-table-association"), &routetableassociation.RouteTableAssociationConfig{
		SubnetId:     privateSubnet.Id(),
		RouteTableId: routeTable.Id(),
	})

	// ECR Repo (You'll need to add awsecr as a dependency)
	// ecrRepo := awsecr.NewRepository(stack, jsii.String("golang-api-repo"), &awsecr.RepositoryConfig{})

	// ECR Image for Golang service (replace with your actual image building logic)
	// You'll likely want to use a CI/CD pipeline to build and push your image
	// ecrImage := awsecr.NewImage(stack, jsii.String("golang-api-image"), &awsecr.ImageConfig{
	// 	RepositoryName: ecrRepo.Name(),
	//  // ... (configure image build and push)
	// })

	// ALB
	alb := alb.NewAlb(stack, jsii.String("nqdi-alb"), &alb.AlbConfig{
		Name:             jsii.String("nqdi-alb"),
		Internal:         jsii.Bool(false),
		LoadBalancerType: jsii.String("application"),
		SecurityGroups:   jsii.Strings(), // You might need to create a security group
		Subnets:          &[]*string{publicSubnet.Id()},
	})

	targetGroup := albtargetgroup.NewAlbTargetGroup(stack, jsii.String("nqdi-alb-target-group"), &albtargetgroup.AlbTargetGroupConfig{
		Name:     jsii.String("nqdi-alb-target-group"),
		Port:     jsii.Number(80),
		Protocol: jsii.String("HTTP"),
		VpcId:    vpc.Id(),
		// Health check configuration
		HealthCheck: &albtargetgroup.AlbTargetGroupHealthCheck{
			Path:               jsii.String("/ping"), // Replace with your health check path
			Protocol:           jsii.String("HTTP"),
			Matcher:            jsii.String("200"),
			Interval:           jsii.Number(30),
			HealthyThreshold:   jsii.Number(2),
			UnhealthyThreshold: jsii.Number(2),
		},
	})

	listener := alblistener.NewAlbListener(stack, jsii.String("nqdi-alb-listener"), &alblistener.AlbListenerConfig{
		LoadBalancerArn: alb.Arn(),
		Port:            jsii.Number(80),
		Protocol:        jsii.String("HTTP"),
		DefaultAction: &[]*alblistener.AlbListenerDefaultAction{
			{Type: jsii.String("forward"),
				TargetGroupArn: targetGroup.Arn()},
		},
	})

	listener.Count()

	// ECS Cluster
	cluster := awsecs.NewEcsCluster(stack, jsii.String("nqdi-ecs-cluster"), &awsecs.EcsClusterConfig{
		Name: jsii.String("nqdi-ecs-cluster"),
	})

	// IAM Role for ECS Task Execution
	executionRole := iamrole.NewIamRole(stack, jsii.String("ecsTaskExecutionRole"), &iamrole.IamRoleConfig{
		Name: jsii.String("ecsTaskExecutionRole"),
		AssumeRolePolicy: jsii.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Action": "sts:AssumeRole",
					"Principal": {
						"Service": "ecs-tasks.amazonaws.com"
					},
					"Effect": "Allow",
					"Sid": ""
				}
			]
		}`),
	})

	iampolicyattachment.NewIamPolicyAttachment(stack, jsii.String("AmazonECSTaskExecutionRolePolicy"), &iampolicyattachment.IamPolicyAttachmentConfig{
		Roles:     &[]*string{executionRole.Name()},
		Name:      jsii.String("nqdi-ecs-exec-policy-attachment"),
		PolicyArn: jsii.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	})

	// IAM Role for ECS Task
	taskRole := iamrole.NewIamRole(stack, jsii.String("ecsTaskRole"), &iamrole.IamRoleConfig{
		Name: jsii.String("ecsTaskRole"),
		AssumeRolePolicy: jsii.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Action": "sts:AssumeRole",
					"Principal": {
						"Service": "ecs-tasks.amazonaws.com"
					},
					"Effect": "Allow",
					"Sid": ""
				}
			]
		}`),
	})

	// (Optional) Add necessary policies to the task role, e.g., for accessing secrets manager
	// awsiam.NewRolePolicyAttachment(stack, jsii.String("AmazonECSTaskRolePolicy"), &awsiam.RolePolicyAttachmentConfig{
	// 	Role:      taskRole.Name(),
	// 	PolicyArn: jsii.String("arn:aws:iam::aws:policy/SecretsManagerReadWrite"),
	// })

	// Fargate Task Definition
	taskDefinition := ecstaskdefinition.NewEcsTaskDefinition(stack, jsii.String("nqdi-fargate-task-definition"), &ecstaskdefinition.EcsTaskDefinitionConfig{
		Family:                  jsii.String("nqdi-rest-api-task"),
		Cpu:                     jsii.String("256"),
		Memory:                  jsii.String("512"),
		NetworkMode:             jsii.String("awsvpc"),
		RequiresCompatibilities: jsii.Strings("FARGATE"),
		ExecutionRoleArn:        executionRole.Arn(),
		TaskRoleArn:             taskRole.Arn(),
		ContainerDefinitions: jsii.String(`[
			{
				"name": "golang-api-container",
				"image": "` + /*ecrImage.ImageUri()*/ "DUDE_REPLACE_WITH_YOUR_IMAGE_URI" + `",
				"portMappings": [
					{
						"containerPort": 80,
						"hostPort": 80
					}
				],
				"environment": [
					{
						"name": "CRDB_CONNECTION_STRING",
						"value": "GET_CRDB_CONNECTION_STRING"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "nqdi-ecs-task-logs",
						"awslogs-region": "eu-central-1",
						"awslogs-stream-prefix": "golang-api"
					}
				}
			}
		]`),
	})

	// ECS Service
	ecsservice.NewEcsService(stack, jsii.String("nqdi-fargate-service"), &ecsservice.EcsServiceConfig{
		Name:           jsii.String("golang-api-service"),
		Cluster:        cluster.Id(),
		TaskDefinition: taskDefinition.Arn(),
		DesiredCount:   jsii.Number(1),
		LaunchType:     jsii.String("FARGATE"),
		NetworkConfiguration: &ecsservice.EcsServiceNetworkConfiguration{
			Subnets:        &[]*string{privateSubnet.Id()},
			AssignPublicIp: jsii.Bool(false),
			SecurityGroups: jsii.Strings(), // You might need to create a security group
		},
		LoadBalancer: &[]*ecsservice.EcsServiceLoadBalancer{
			{
				TargetGroupArn: targetGroup.Arn(),
				ContainerName:  jsii.String("golang-api-container"),
				ContainerPort:  jsii.Number(80),
			},
		},
	})

	// let's leave route53 until later on, hey?

	// Route53 (replace with your actual domain and hosted zone ID)
	// zone := awsroute53.NewZone(stack, jsii.String("nqdi-route53-zone"), &awsroute53.ZoneConfig{
	// 	Name: jsii.String("example.com"), // Replace with your domain
	// 	// ... (configure other zone settings)
	// })

	// awsroute53.NewRecord(stack, jsii.String("nqdi-route53-record"), &awsroute53.RecordConfig{
	// 	Name: jsii.String("api"),
	// 	Type: jsii.String("A"),
	// 	Zone: zone.Id(),
	// 	Aliases: &[]*awsroute53.RecordAlias{{
	// 		Name:                 alb.DnsName(),
	// 		Zone:                 alb.ZoneId(),
	// 		EvaluateTargetHealth: jsii.Bool(true),
	// 	}},
	// })

	// // ACM (optional, for HTTPS)
	// // Replace with your actual domain name
	// certificate := awsacm.NewCertificate(stack, jsii.String("nqdi-acm-certificate"), &awsacm.CertificateConfig{
	// 	DomainName:       jsii.String("api.example.com"),
	// 	ValidationMethod: jsii.String("DNS"),
	// })

	// ... (configure DNS validation for the certificate)

	// Outputs
	cdktf.NewTerraformOutput(stack, jsii.String("alb_dns_name"), &cdktf.TerraformOutputConfig{
		Value: alb.DnsName(),
	})

	return stack
}

func main() {
	app := cdktf.NewApp(nil)
	// stack := RestApiInfraFargate(app, "rest_api_fargate")
	stack := CockroachDbTest(app, "cockroachdb_test")
	cdktf.NewRemoteBackend(stack, &cdktf.RemoteBackendConfig{
		Hostname:     jsii.String("app.terraform.io"),
		Organization: jsii.String("nqdi"),
		Workspaces:   cdktf.NewNamedRemoteWorkspace(jsii.String("rest-api-infra")),
	})

	app.Synth()
}
