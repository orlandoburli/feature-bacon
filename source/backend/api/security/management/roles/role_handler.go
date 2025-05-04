package roles

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orlandoburli/feature-bacon/api/persistence"
	"net/http"
)

func GetRoles(c *gin.Context) {
	var roles []Role

	db := persistence.ConnectDb()

	db.Find(&roles)

	c.JSON(http.StatusOK, roles)
}

func GetRole(c *gin.Context) {
	var role Role

	idString := c.Param("id")

	id, err := uuid.Parse(idString)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := persistence.ConnectDb()

	if err := db.Where("id = ?", id).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

func CreateRole(c *gin.Context) {
	var input CreateRoleRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	role := Role{Name: input.Name}

	persistence.ConnectDb().Create(&role)

	c.JSON(http.StatusCreated, role)
}

func UpdateRole(c *gin.Context) {
	var input UpdateRoleRequest

	idString := c.Param("id")

	id, err := uuid.Parse(idString)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var role Role

	db := persistence.ConnectDb()

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	}

	if err = db.Where("id = ?", id).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	role.Name = input.Name

	db.Save(&role)

	c.JSON(http.StatusOK, role)
}

func DeleteRole(c *gin.Context) {
	idString := c.Param("id")

	id, err := uuid.Parse(idString)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var role Role

	db := persistence.ConnectDb()

	if err := db.Where("id = ?", id).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	db.Delete(&role)

	c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
}
