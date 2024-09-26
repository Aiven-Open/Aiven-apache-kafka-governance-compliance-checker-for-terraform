package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func InitMicroserviceServer() {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	// Endpoints
	e.POST("/check", check)
	e.POST("/proxytoaiven", proxyTokenToAiven)

	e.Logger.Fatal(e.Start(":1323"))
}

type CheckRequestBody struct {
	Requester string `json:"requester"`
	Approvers string `json:"approvers"`
	Plan      Plan   `json:"plan"`
}

func check(c echo.Context) error {
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
}

type GovernanceConfigResponse struct {
	CreateTime                   string `json:"create_time"`
	CreatedBy                    string `json:"created_by"`
	DefaultGovernanceUserGroupId string `json:"default_governance_user_group_id"`
	DefaultPartitions            uint16 `json:"default_partitions"`
	DefaultReplicationFactor     uint16 `json:"default_replication_factor"`
	GlobalMembershipRequirement  string `json:"global_membership_requirement"`
	KafkaGovernanceEnabled       bool   `json:"kafka_governance_enabled"`
	MaxPartitions                uint16 `json:"max_partitions"`
	MaxReplicationFactor         uint16 `json:"max_replication_factor"`
	UpdateTime                   string `json:"update_time"`
	UpdatedBy                    string `json:"updated_by"`
}

func proxyTokenToAiven(c echo.Context) error {
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	// Call the api
	fmt.Println("call Aiven API")
	orgId := c.QueryParam("org")
	url := "https://console.aiven.io/v1/organization/org" + orgId + "/governance/config"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", c.Request().Header.Get("Authorization")) // pass the token through

	resp, err := client.Do(req)

	// handle errors
	if err != nil {
		fmt.Println("got an error ", err)
		return err
	}

	// handle non-OK status codes
	if resp.StatusCode != http.StatusOK {
		return c.JSON(resp.StatusCode, resp.Status)
	}

	// Unmarshal the response into a GovernanceConfigResponse struct
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var govConfigResponse GovernanceConfigResponse
	if err = json.Unmarshal(body, &govConfigResponse); err != nil {
		return err
	}

	// we could do something useful with the information here but instead we just return it

	return c.JSON(http.StatusOK, govConfigResponse)
}
