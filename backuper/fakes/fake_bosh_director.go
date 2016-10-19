// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
)

type FakeBoshDirector struct {
	FindInstancesStub        func(deploymentName string) (backuper.Instances, error)
	findInstancesMutex       sync.RWMutex
	findInstancesArgsForCall []struct {
		deploymentName string
	}
	findInstancesReturns struct {
		result1 backuper.Instances
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeBoshDirector) FindInstances(deploymentName string) (backuper.Instances, error) {
	fake.findInstancesMutex.Lock()
	fake.findInstancesArgsForCall = append(fake.findInstancesArgsForCall, struct {
		deploymentName string
	}{deploymentName})
	fake.recordInvocation("FindInstances", []interface{}{deploymentName})
	fake.findInstancesMutex.Unlock()
	if fake.FindInstancesStub != nil {
		return fake.FindInstancesStub(deploymentName)
	} else {
		return fake.findInstancesReturns.result1, fake.findInstancesReturns.result2
	}
}

func (fake *FakeBoshDirector) FindInstancesCallCount() int {
	fake.findInstancesMutex.RLock()
	defer fake.findInstancesMutex.RUnlock()
	return len(fake.findInstancesArgsForCall)
}

func (fake *FakeBoshDirector) FindInstancesArgsForCall(i int) string {
	fake.findInstancesMutex.RLock()
	defer fake.findInstancesMutex.RUnlock()
	return fake.findInstancesArgsForCall[i].deploymentName
}

func (fake *FakeBoshDirector) FindInstancesReturns(result1 backuper.Instances, result2 error) {
	fake.FindInstancesStub = nil
	fake.findInstancesReturns = struct {
		result1 backuper.Instances
		result2 error
	}{result1, result2}
}

func (fake *FakeBoshDirector) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.findInstancesMutex.RLock()
	defer fake.findInstancesMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeBoshDirector) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ backuper.BoshDirector = new(FakeBoshDirector)
