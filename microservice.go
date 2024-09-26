package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CheckRequestBody struct {
	Requester string `json:"requester"`
	Approvers string `json:"approvers"`
	Plan      Plan   `json:"plan"`
}

func InitMicroserviceServer() {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	e.POST("/check", func(c echo.Context) error {
		fmt.Println("Got a request")

		var req CheckRequestBody
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		result := CheckPlan(&req.Requester, &req.Approvers, req.Plan)
		if result.Ok {
			return c.JSON(http.StatusOK, result)
		} else {
			return c.JSON(http.StatusTeapot, result)
		}
	})

	e.Logger.Info(e.Start(":1323"))

	fmt.Println("Aiven Terraform governance compliance checker as a microservice ready to process requests!")
}
