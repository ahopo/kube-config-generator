package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"gopkg.in/yaml.v2"
)

type kubeConfig struct {
	APIVersion  string     `yaml:"apiVersion"`
	Clusters    []cluster  `yaml:"clusters"`
	Contexts    []context2 `yaml:"contexts"`
	CurrentCtx  string     `yaml:"current-context"`
	Kind        string     `yaml:"kind"`
	Preferences struct{}   `yaml:"preferences"`
	Users       []user     `yaml:"users"`
}

type cluster struct {
	Cluster clusterData `yaml:"cluster"`
	Name    string      `yaml:"name"`
}

type clusterData struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	Server                   string `yaml:"server"`
}

type context2 struct {
	Context contextData `yaml:"context"`
	Name    string      `yaml:"name"`
}

type contextData struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type user struct {
	Name string   `yaml:"name"`
	User userData `yaml:"user"`
}

type userData struct {
	Exec execData `yaml:"exec"`
}

type execData struct {
	APIVersion string   `yaml:"apiVersion"`
	Command    string   `yaml:"command"`
	Args       []string `yaml:"args"`
}

func main() {
	ClusterName := ""
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println(err)
	}

	eksClient := eks.NewFromConfig(cfg)

	clusterName := ClusterName

	describeClusterOutput, err := eksClient.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		fmt.Println(err)
	}

	endpoint := aws.ToString(describeClusterOutput.Cluster.Endpoint)
	caData := aws.ToString(describeClusterOutput.Cluster.CertificateAuthority.Data)
	clusterArn := aws.ToString(describeClusterOutput.Cluster.Arn)

	kubeConfigPath := fmt.Sprintf("%s-config", ClusterName)

	kubeConfigContent := kubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []cluster{
			{
				Name: clusterArn,
				Cluster: clusterData{
					CertificateAuthorityData: caData,
					Server:                   endpoint,
				},
			},
		},
		Contexts: []context2{
			{
				Name: clusterArn,
				Context: contextData{
					Cluster: clusterArn,
					User:    clusterArn,
				},
			},
		},
		CurrentCtx:  clusterArn,
		Preferences: struct{}{},
		Users: []user{
			{
				Name: clusterArn,
				User: userData{
					Exec: execData{
						APIVersion: "client.authentication.k8s.io/v1beta1",
						Args: []string{
							"--region",
							cfg.Region,
							"eks",
							"get-token",
							"--cluster-name",
							clusterName,
						},
						Command: "aws",
					},
				},
			},
		},
	}

	kubeConfigFile, err := os.Create(kubeConfigPath)
	if err != nil {
		fmt.Println("Failed to create kubeConfig file:", err)
		fmt.Println(err)
	}

	encoder := yaml.NewEncoder(kubeConfigFile)
	defer encoder.Close()

	err = encoder.Encode(&kubeConfigContent)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Kube Config file created at %s\n", kubeConfigPath)

}
