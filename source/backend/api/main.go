package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/orlandoburli/feature-bacon/api/persistence"
	"github.com/orlandoburli/feature-bacon/api/security/management/roles"
)

func main() {
	initDB()

	r := gin.Default()

	mapHandlersToRoutes(r)

	err := r.Run()

	if err != nil {
		fmt.Printf("Error starting server %s\n", err)
		return
	}
}

func initDB() {
	db := persistence.ConnectDb()

	err := persistence.AddExtraFunctions()

	if err != nil {
		panic("Failed to add extra functions")
	}

	err = db.AutoMigrate(&roles.Role{})

	if err != nil {
		panic("failed to execute migration")
	}
}

func mapHandlersToRoutes(r *gin.Engine) {
	r.GET("/security/management/roles", roles.GetRoles)
	r.GET("/security/management/roles/:id", roles.GetRole)
	r.POST("/security/management/roles", roles.CreateRole)
	r.PUT("/security/management/roles/:id", roles.UpdateRole)
	r.DELETE("/security/management/roles/:id", roles.DeleteRole)
}
