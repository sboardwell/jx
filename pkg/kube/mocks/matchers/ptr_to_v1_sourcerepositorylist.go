// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/petergtz/pegomock"
	"reflect"
)

func AnyPtrToV1SourceRepositoryList() *v1.SourceRepositoryList {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(*v1.SourceRepositoryList))(nil)).Elem()))
	var nullValue *v1.SourceRepositoryList
	return nullValue
}

func EqPtrToV1SourceRepositoryList(value *v1.SourceRepositoryList) *v1.SourceRepositoryList {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue *v1.SourceRepositoryList
	return nullValue
}
