package cloud_provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	"k8s.io/cloud-provider-baiducloud/pkg/cloud-sdk/blb"
)

var errBlbNotExist = errors.New(" BLB does not exist during ensuring BLB! ")

func (bc *Baiducloud) ensureBLB(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node, serviceAnnotation *ServiceAnnotation) (*blb.LoadBalancer, error) {
	var lb *blb.LoadBalancer
	var err error

	//before checking CceAutoAddLoadBalancerId checks LoadBalancerExistId firstly
	// if serviceAnnotation.CceAutoAddLoadBalancerId is none, we need to double check from cloud since user can update yaml in a short time causing annotation not attach.
	// LOG:
	// Event(v1.ObjectReference{Kind:"Service", Namespace:"default", Name:"eip-lb-service5",
	// UID:"887fc095-3b25-11e9-8eab-fa163e80f2ac", APIVersion:"v1", ResourceVersion:"172548", FieldPath:""}): type: 'Warning'
	// reason: 'CreatingLoadBalancerFailed' Error creating load balancer (will retry): not persisting update to service
	// 'default/eip-lb-service5' that has been changed since we received it: Operation cannot be fulfilled on services
	// "eip-lb-service5": the object has been modified; please apply your changes to the latest version and try again
	if len(serviceAnnotation.LoadBalancerExistId) == 0 { //do not use annotations LoadBalancerExistId
		if len(serviceAnnotation.CceAutoAddLoadBalancerId) == 0 {
			//get blb according to the clusterid+service
			var exists bool
			lb, exists, err = bc.getBCELoadBalancer(bc.ClusterID + "/" + getServiceName(service))
			if err != nil {
				return nil, err
			}
			if exists {
				glog.V(3).Infof("[%v %v] EnsureLoadBalancer serviceAnnotation.CceAutoAddLoadBalancerId is none, but we got blb from cloud", service.Namespace, service.Name)
				serviceAnnotation.CceAutoAddLoadBalancerId = lb.BlbId
				if service.Annotations == nil {
					service.Annotations = make(map[string]string)
				}
				service.Annotations[ServiceAnnotationCceAutoAddLoadBalancerId] = lb.BlbId
			}
		}

		if len(serviceAnnotation.CceAutoAddLoadBalancerId) == 0 { // blb not exist, create one and update annotation
			//create a blb
			loadbalancerid, err := bc.createBLB(service, serviceAnnotation)
			if err != nil {
				return nil, err
			}
			//get the status of the blb
			lbs, err := bc.describeBLB(loadbalancerid, service)
			if err != nil {
				return nil, err
			}
			//add blbid on the annotation
			lb = &lbs[0]
			if service.Annotations == nil {
				service.Annotations = make(map[string]string)
			}
			service.Annotations[ServiceAnnotationCceAutoAddLoadBalancerId] = lb.BlbId
		} else if lb == nil { // blb already exist, get info from cloud
			var exists bool
			lb, exists, err = bc.getBCELoadBalancerById(serviceAnnotation.CceAutoAddLoadBalancerId)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, fmt.Errorf("EnsureLoadBalancer getBCELoadBalancerById failed, target blb not exist, blb id: %v", serviceAnnotation.CceAutoAddLoadBalancerId)
			}
			glog.V(3).Infof("[%v %v] EnsureLoadBalancer: blb already exists: %v", service.Namespace, service.Name, lb)
		}

	} else { //use annotations LoadBalancerExistId
		//blb has been used.
		var exists bool
		lb, exists, err = bc.getBCELoadBalancerById(serviceAnnotation.LoadBalancerExistId)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("EnsureLoadBalancer getBCELoadBalancerById failed, target blb not exist, blb id: %v", serviceAnnotation.CceAutoAddLoadBalancerId)
		}
		allListeners, err := bc.getAllListeners(lb)
		if err != nil {
			return nil, err
		}

		if len(allListeners) > 0 && len(serviceAnnotation.CceAutoAddLoadBalancerId) == 0 {
			service.Annotations[ServiceAnnotationLoadBalancerExistId] = "error_blb_has_been_used"
			return nil, fmt.Errorf(" This blb has been used already! ")
		} else {
			glog.V(3).Infof("[%v %v] EnsureLoadBalancerexistid: blb already exists: %v", service.Namespace, service.Name, lb)
			service.Annotations[ServiceAnnotationCceAutoAddLoadBalancerId] = serviceAnnotation.LoadBalancerExistId
		}
	}

	lb, err = bc.waitForLoadBalancer(lb)
	if err != nil {
		bc.dealWithError(err, service)
		return nil, err
	}

	// update listener
	glog.V(2).Infof("[%v %v] EnsureLoadBalancer: reconcileListeners!", service.Namespace, service.Name)
	err = bc.reconcileListeners(service, lb)
	if err != nil {
		return nil, err
	}
	lb, err = bc.waitForLoadBalancer(lb)
	if err != nil {
		bc.dealWithError(err, service)
		return nil, err
	}
	// update backend server
	glog.V(2).Infof("[%v %v] EnsureLoadBalancer: reconcileBackendServers!", service.Namespace, service.Name)
	err = bc.reconcileBackendServers(service, nodes, lb)
	if err != nil {
		return nil, err
	}
	lb, err = bc.waitForLoadBalancer(lb)
	if err != nil {
		bc.dealWithError(err, service)
		return nil, err
	}
	return lb, nil
}
func (bc *Baiducloud) createBLB(service *v1.Service, serviceAnnotation *ServiceAnnotation) (string, error) {
	glog.V(3).Infof("[%v %v] EnsureLoadBalancer create blb!", service.Namespace, service.Name)
	vpcID, subnetID, err := bc.getVpcInfoForBLB(serviceAnnotation)
	if err != nil {
		return "", fmt.Errorf(" Can't get VPC info for BLB: %v\n ", err)
	}

	allocateVip := false
	if serviceAnnotation.LoadBalancerAllocateVip == "true" {
		allocateVip = true
	}
	blbName := bc.ClusterID + "/" + getServiceName(service)
	// according to blb api doc, limit to 65
	if len(blbName) > 65 {
		blbName = blbName[:65]
	}
	args := blb.CreateLoadBalancerArgs{
		Name:        blbName,
		VpcID:       vpcID,
		SubnetID:    subnetID,
		Desc:        "auto generated by cce:" + bc.ClusterID,
		AllocateVip: allocateVip,
	}
	glog.V(3).Infof("[%v %v] EnsureLoadBalancer create blb args: %v", service.Namespace, service.Name, args)
	resp, err := bc.clientSet.Blb().CreateLoadBalancer(&args)
	if err != nil {
		return "", err
	}
	glog.V(3).Infof("[%v %v] EnsureLoadBalancer create blb success, BLB name: %s, BLB id: %s, BLB address: %s.", service.Namespace, service.Name, resp.Name, resp.LoadBalancerId, resp.Address)
	return resp.LoadBalancerId, nil
}

func (bc *Baiducloud) describeBLB(loadbalancerid string, service *v1.Service) ([]blb.LoadBalancer, error) {
	argsDesc := blb.DescribeLoadBalancersArgs{
		LoadBalancerId: loadbalancerid,
		ExactlyMatch:   true,
	}
	lbs, err := bc.clientSet.Blb().DescribeLoadBalancers(&argsDesc)
	if err != nil {
		return nil, err
	}
	if len(lbs) != 1 {
		tryCount := 0
		for {
			tryCount++
			if tryCount > 10 {
				return nil, fmt.Errorf("EnsureLoadBalancer create blb success but query get none")
			}
			glog.V(3).Infof("[%v %v] EnsureLoadBalancer create blb success but query get none, tryCount: %v", service.Namespace, service.Name, tryCount)
			lbs, err = bc.clientSet.Blb().DescribeLoadBalancers(&argsDesc)
			if err != nil {
				return nil, err
			}
			if len(lbs) == 1 {
				glog.V(3).Infof("[%v %v] EnsureLoadBalancer create blb success and query get one, tryCount: %v", service.Namespace, service.Name, tryCount)
				break
			}
			time.Sleep(10 * time.Second)
		}
	}
	return lbs, nil
}

func (bc *Baiducloud) waitForLoadBalancer(lb *blb.LoadBalancer) (*blb.LoadBalancer, error) {
	lb.Status = "unknown" // add here to do loop
	for index := 0; (index < 10) && (lb.Status != "available"); index++ {
		glog.V(3).Infof("BLB: %s is not available, retry:  %d", lb.BlbId, index)
		time.Sleep(10 * time.Second)
		newlb, exist, err := bc.getBCELoadBalancerById(lb.BlbId)
		if err != nil {
			glog.V(3).Infof("getBCELoadBalancer error: %s", lb.BlbId)
			return newlb, err
		}
		if !exist {
			glog.V(3).Infof("getBCELoadBalancer not exist: %s, retry", lb.BlbId)
			if index >= 9 {
				err = errBlbNotExist
				return newlb, err
			}
			continue
		}
		lb = newlb
		glog.V(3).Infof("BLB status is : %s", lb.Status)
		if index >= 9 && lb.Status != "available" {
			return nil, fmt.Errorf("waitForLoadBalancer failed after retry")
		}
	}
	return lb, nil
}

func (bc *Baiducloud) dealWithError(err error, service *v1.Service) {
	switch err {
	case errBlbNotExist:
		if service.Annotations != nil {
			delete(service.Annotations, ServiceAnnotationCceAutoAddLoadBalancerId)
		}
	default:
	}
}
