/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloud_provider

import (
	"strconv"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

const (
	// ServiceAnnotationLoadBalancerPrefix is the annotation prefix of LoadBalancer
	ServiceAnnotationLoadBalancerPrefix = "service.beta.kubernetes.io/cce-load-balancer-"
	// ServiceAnnotationLoadBalancerId is the annotation of LoadBalancerId
	ServiceAnnotationLoadBalancerId = ServiceAnnotationLoadBalancerPrefix + "id"
	// ServiceAnnotationLoadBalancerInternalVpc is the annotation of LoadBalancerInternalVpc
	ServiceAnnotationLoadBalancerInternalVpc = ServiceAnnotationLoadBalancerPrefix + "internal-vpc"
	// ServiceAnnotationLoadBalancerAllocateVip is the annotation which indicates BLB with a VIP
	ServiceAnnotationLoadBalancerAllocateVip = ServiceAnnotationLoadBalancerPrefix + "allocate-vip"
	// TODO:
	// ServiceAnnotationLoadBalancerScheduler is the annotation of load balancer which can be "RoundRobin"/"LeastConnection"/"Hash"
	ServiceAnnotationLoadBalancerScheduler = ServiceAnnotationLoadBalancerPrefix + "scheduler"
	// TODO:
	// ServiceAnnotationLoadBalancerHealthCheckTimeoutInSecond is the annotation of health check timeout, default 3s, [1, 60]
	ServiceAnnotationLoadBalancerHealthCheckTimeoutInSecond = ServiceAnnotationLoadBalancerPrefix + "health-check-timeout-in-second"
	// TODO:
	// ServiceAnnotationLoadBalancerHealthCheckInterval is the annotation of health check interval, default 3s, [1, 10]
	ServiceAnnotationLoadBalancerHealthCheckInterval = ServiceAnnotationLoadBalancerPrefix + "health-check-interval"
	// TODO:
	// ServiceAnnotationLoadBalancerUnhealthyThreshold is the annotation of unhealthy threshold, default 3, [2, 5]
	ServiceAnnotationLoadBalancerUnhealthyThreshold = ServiceAnnotationLoadBalancerPrefix + "unhealthy-threshold"
	// TODO:
	// ServiceAnnotationLoadBalancerHealthyThreshold is the annotation of healthy threshold, default 3, [2, 5]
	ServiceAnnotationLoadBalancerHealthyThreshold = ServiceAnnotationLoadBalancerPrefix + "healthy-threshold"
	// TODO:
	// ServiceAnnotationLoadBalancerHealthCheckString is the annotation of health check string
	ServiceAnnotationLoadBalancerHealthCheckString = ServiceAnnotationLoadBalancerPrefix + "health-check-string"

	// ServiceAnnotationElasticIPPrefix is the annotation prefix of ElasticIP
	ServiceAnnotationElasticIPPrefix = "service.beta.kubernetes.io/cce-elastic-ip-"
	// ServiceAnnotationElasticIPName is the annotation of ElasticIPName
	ServiceAnnotationElasticIPName = ServiceAnnotationElasticIPPrefix + "name"
	// ServiceAnnotationElasticIPPaymentTiming is the annotation of ElasticIPPaymentTiming
	ServiceAnnotationElasticIPPaymentTiming = ServiceAnnotationElasticIPPrefix + "payment-timing"
	// ServiceAnnotationElasticIPBillingMethod is the annotation of ElasticIPBillingMethod
	ServiceAnnotationElasticIPBillingMethod = ServiceAnnotationElasticIPPrefix + "billing-method"
	// ServiceAnnotationElasticIPBandwidthInMbps is the annotation of ElasticIPBandwidthInMbps
	ServiceAnnotationElasticIPBandwidthInMbps = ServiceAnnotationElasticIPPrefix + "bandwidth-in-mbps"
	// ServiceAnnotationElasticIPReservationLength is the annotation of ElasticIPReservationLength
	ServiceAnnotationElasticIPReservationLength = ServiceAnnotationElasticIPPrefix + "reservation-length"
)

const (
	// NodeAnnotationPrefix is the annotation prefix of Node
	NodeAnnotationPrefix = "node.alpha.kubernetes.io/"
	// NodeAnnotationVpcId is the annotation of VpcId on node
	NodeAnnotationVpcId = NodeAnnotationPrefix + "vpc-id"
	// NodeAnnotationVpcRouteTableId is the annotation of VpcRouteTableId on node
	NodeAnnotationVpcRouteTableId = NodeAnnotationPrefix + "vpc-route-table-id"
	// NodeAnnotationVpcRouteRuleId is the annotation of VpcRouteRuleId on node
	NodeAnnotationVpcRouteRuleId = NodeAnnotationPrefix + "vpc-route-rule-id"

	// NodeAnnotationCCMVersion is the version of CCM
	NodeAnnotationCCMVersion = NodeAnnotationPrefix + "ccm-version"
)

// ServiceAnnotation contains annotations from service
type ServiceAnnotation struct {
	/* BLB */
	LoadBalancerId                         string
	LoadBalancerInternalVpc                string
	LoadBalancerAllocateVip                string
	LoadBalancerScheduler                  string
	LoadBalancerHealthCheckTimeoutInSecond int
	LoadBalancerHealthCheckInterval        int
	LoadBalancerUnhealthyThreshold         int
	LoadBalancerHealthyThreshold           int
	LoadBalancerHealthCheckString          string

	/* EIP */
	ElasticIPName              string
	ElasticIPPaymentTiming     string
	ElasticIPBillingMethod     string
	ElasticIPBandwidthInMbps   int
	ElasticIPReservationLength int
}

// NodeAnnotation contains annotations from node
type NodeAnnotation struct {
	VpcId           string
	VpcRouteTableId string
	VpcRouteRuleId  string
	CCMVersion      string
}

// ExtractServiceAnnotation extract annotations from service
func ExtractServiceAnnotation(service *v1.Service) *ServiceAnnotation {
	glog.V(4).Infof("start to ExtractServiceAnnotation: %v", service.Annotations)
	result := &ServiceAnnotation{}
	annotation := make(map[string]string)
	for k, v := range service.Annotations {
		annotation[k] = v
	}

	loadBalancerId, exist := annotation[ServiceAnnotationLoadBalancerId]
	if exist {
		result.LoadBalancerId = loadBalancerId
	}

	loadBalancerInternalVpc, exist := annotation[ServiceAnnotationLoadBalancerInternalVpc]
	if exist {
		result.LoadBalancerInternalVpc = loadBalancerInternalVpc
	}

	loadBalancerAllocateVip, ok := annotation[ServiceAnnotationLoadBalancerAllocateVip]
	if ok {
		result.LoadBalancerAllocateVip = loadBalancerAllocateVip
	}

	loadBalancerScheduler, ok := annotation[ServiceAnnotationLoadBalancerScheduler]
	if ok {
		result.LoadBalancerScheduler = loadBalancerScheduler
	}

	loadBalancerHealthCheckTimeoutInSecond, exist := annotation[ServiceAnnotationLoadBalancerHealthCheckTimeoutInSecond]
	if exist {
		i, err := strconv.Atoi(loadBalancerHealthCheckTimeoutInSecond)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationLoadBalancerHealthCheckTimeoutInSecond must be int")
		} else {
			result.LoadBalancerHealthCheckTimeoutInSecond = i
		}
	}

	loadBalancerHealthCheckInterval, exist := annotation[ServiceAnnotationLoadBalancerHealthCheckInterval]
	if exist {
		i, err := strconv.Atoi(loadBalancerHealthCheckInterval)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationLoadBalancerHealthCheckInterval must be int")
		} else {
			result.LoadBalancerHealthCheckInterval = i
		}
	}

	loadBalancerUnhealthyThreshold, exist := annotation[ServiceAnnotationLoadBalancerUnhealthyThreshold]
	if exist {
		i, err := strconv.Atoi(loadBalancerUnhealthyThreshold)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationLoadBalancerUnhealthyThreshold must be int")
		} else {
			result.LoadBalancerUnhealthyThreshold = i
		}
	}

	loadBalancerHealthyThreshold, exist := annotation[ServiceAnnotationLoadBalancerHealthyThreshold]
	if exist {
		i, err := strconv.Atoi(loadBalancerHealthyThreshold)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationLoadBalancerHealthyThreshold must be int")
		} else {
			result.LoadBalancerHealthyThreshold = i
		}
	}

	loadBalancerHealthCheckString, exist := annotation[ServiceAnnotationLoadBalancerHealthCheckString]
	if exist {
		result.LoadBalancerHealthCheckString = loadBalancerHealthCheckString
	}

	elasticIPName, exist := annotation[ServiceAnnotationElasticIPName]
	if exist {
		result.ElasticIPName = elasticIPName
	}

	elasticIPPaymentTiming, exist := annotation[ServiceAnnotationElasticIPPaymentTiming]
	if exist {
		result.ElasticIPPaymentTiming = elasticIPPaymentTiming
	}

	elasticIPBillingMethod, exist := annotation[ServiceAnnotationElasticIPBillingMethod]
	if exist {
		result.ElasticIPBillingMethod = elasticIPBillingMethod
	}

	elasticIPBandwidthInMbps, exist := annotation[ServiceAnnotationElasticIPBandwidthInMbps]
	if exist {
		i, err := strconv.Atoi(elasticIPBandwidthInMbps)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationElasticIPBandwidthInMbps must be int")
		} else {
			result.ElasticIPBandwidthInMbps = i
		}
	}

	elasticIPReservationLength, exist := annotation[ServiceAnnotationElasticIPReservationLength]
	if exist {
		i, err := strconv.Atoi(elasticIPReservationLength)
		if err != nil {
			glog.V(4).Infof("ServiceAnnotationElasticIPReservationLength must be int")
		} else {
			result.ElasticIPReservationLength = i
		}
	}

	return result
}

// ExtractNodeAnnotation extract annotations from node
func ExtractNodeAnnotation(node *v1.Node) *NodeAnnotation {
	glog.V(4).Infof("start to ExtractNodeAnnotation: %v", node.Annotations)
	result := &NodeAnnotation{}
	annotation := make(map[string]string)
	for k, v := range node.Annotations {
		annotation[k] = v
	}

	vpcId, ok := annotation[NodeAnnotationVpcId]
	if ok {
		result.VpcId = vpcId
	}

	vpcRouteTableId, ok := annotation[NodeAnnotationVpcRouteTableId]
	if ok {
		result.VpcRouteTableId = vpcRouteTableId
	}

	vpcRouteRuleId, ok := annotation[NodeAnnotationVpcRouteRuleId]
	if ok {
		result.VpcRouteRuleId = vpcRouteRuleId
	}

	ccmVersion, ok := annotation[NodeAnnotationCCMVersion]
	if ok {
		result.CCMVersion = ccmVersion
	}

	return result
}
