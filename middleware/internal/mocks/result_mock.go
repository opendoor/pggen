// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"database/sql"
	"sync"
)

type Result struct {
	LastInsertIdStub        func() (int64, error)
	lastInsertIdMutex       sync.RWMutex
	lastInsertIdArgsForCall []struct {
	}
	lastInsertIdReturns struct {
		result1 int64
		result2 error
	}
	lastInsertIdReturnsOnCall map[int]struct {
		result1 int64
		result2 error
	}
	RowsAffectedStub        func() (int64, error)
	rowsAffectedMutex       sync.RWMutex
	rowsAffectedArgsForCall []struct {
	}
	rowsAffectedReturns struct {
		result1 int64
		result2 error
	}
	rowsAffectedReturnsOnCall map[int]struct {
		result1 int64
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Result) LastInsertId() (int64, error) {
	fake.lastInsertIdMutex.Lock()
	ret, specificReturn := fake.lastInsertIdReturnsOnCall[len(fake.lastInsertIdArgsForCall)]
	fake.lastInsertIdArgsForCall = append(fake.lastInsertIdArgsForCall, struct {
	}{})
	fake.recordInvocation("LastInsertId", []interface{}{})
	fake.lastInsertIdMutex.Unlock()
	if fake.LastInsertIdStub != nil {
		return fake.LastInsertIdStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.lastInsertIdReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *Result) LastInsertIdCallCount() int {
	fake.lastInsertIdMutex.RLock()
	defer fake.lastInsertIdMutex.RUnlock()
	return len(fake.lastInsertIdArgsForCall)
}

func (fake *Result) LastInsertIdCalls(stub func() (int64, error)) {
	fake.lastInsertIdMutex.Lock()
	defer fake.lastInsertIdMutex.Unlock()
	fake.LastInsertIdStub = stub
}

func (fake *Result) LastInsertIdReturns(result1 int64, result2 error) {
	fake.lastInsertIdMutex.Lock()
	defer fake.lastInsertIdMutex.Unlock()
	fake.LastInsertIdStub = nil
	fake.lastInsertIdReturns = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *Result) LastInsertIdReturnsOnCall(i int, result1 int64, result2 error) {
	fake.lastInsertIdMutex.Lock()
	defer fake.lastInsertIdMutex.Unlock()
	fake.LastInsertIdStub = nil
	if fake.lastInsertIdReturnsOnCall == nil {
		fake.lastInsertIdReturnsOnCall = make(map[int]struct {
			result1 int64
			result2 error
		})
	}
	fake.lastInsertIdReturnsOnCall[i] = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *Result) RowsAffected() (int64, error) {
	fake.rowsAffectedMutex.Lock()
	ret, specificReturn := fake.rowsAffectedReturnsOnCall[len(fake.rowsAffectedArgsForCall)]
	fake.rowsAffectedArgsForCall = append(fake.rowsAffectedArgsForCall, struct {
	}{})
	fake.recordInvocation("RowsAffected", []interface{}{})
	fake.rowsAffectedMutex.Unlock()
	if fake.RowsAffectedStub != nil {
		return fake.RowsAffectedStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.rowsAffectedReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *Result) RowsAffectedCallCount() int {
	fake.rowsAffectedMutex.RLock()
	defer fake.rowsAffectedMutex.RUnlock()
	return len(fake.rowsAffectedArgsForCall)
}

func (fake *Result) RowsAffectedCalls(stub func() (int64, error)) {
	fake.rowsAffectedMutex.Lock()
	defer fake.rowsAffectedMutex.Unlock()
	fake.RowsAffectedStub = stub
}

func (fake *Result) RowsAffectedReturns(result1 int64, result2 error) {
	fake.rowsAffectedMutex.Lock()
	defer fake.rowsAffectedMutex.Unlock()
	fake.RowsAffectedStub = nil
	fake.rowsAffectedReturns = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *Result) RowsAffectedReturnsOnCall(i int, result1 int64, result2 error) {
	fake.rowsAffectedMutex.Lock()
	defer fake.rowsAffectedMutex.Unlock()
	fake.RowsAffectedStub = nil
	if fake.rowsAffectedReturnsOnCall == nil {
		fake.rowsAffectedReturnsOnCall = make(map[int]struct {
			result1 int64
			result2 error
		})
	}
	fake.rowsAffectedReturnsOnCall[i] = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *Result) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.lastInsertIdMutex.RLock()
	defer fake.lastInsertIdMutex.RUnlock()
	fake.rowsAffectedMutex.RLock()
	defer fake.rowsAffectedMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Result) recordInvocation(key string, args []interface{}) {
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

var _ sql.Result = new(Result)
