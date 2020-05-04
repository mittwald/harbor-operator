package controller

import (
	"github.com/mittwald/harbor-operator/pkg/controller/repository"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, repository.Add)
}
