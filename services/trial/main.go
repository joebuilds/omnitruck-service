package trial

import (
	"sync"
	"time"

	_ "github.com/chef/omnitruck-service/docs/trial"
	omnitruck "github.com/chef/omnitruck-service/omnitruck-client"
	"github.com/chef/omnitruck-service/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

type TrialService struct {
	services.ApiService
}

func NewServer(c services.Config) *TrialService {
	service := TrialService{}
	service.Initialize(c)

	service.Log.Info("Adding EOL Validator")
	eolversion := omnitruck.EolVersionValidator{}
	service.Validator.Add(&eolversion)

	return &service
}

// @title			Licensed Trial Omnitruck API
// @version			1.0
// @description 	Licensed Trial Omnitruck API
// @license.name	Apache 2.0
// @license.url 	http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:3001
func (server *TrialService) Start(wg *sync.WaitGroup) error {
	server.Lock()
	defer server.Unlock()

	server.App = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		EnablePrintRoutes:     false,
		ReadTimeout:           300 * time.Second,
		WriteTimeout:          300 * time.Second,
	})

	server.App.Use(cors.New())
	// This will catch panics in the app and prevent it from crashing the server
	// TODO: Figure out if we can better handle logging these, currently it just returns a panic message to the user
	// server.App.Use(recover.New())
	server.buildRouter()

	wg.Add(1)
	go server.StartService()

	return nil
}

func (server *TrialService) HealthCheck(c *fiber.Ctx) error {
	res := map[string]interface{}{
		"data": "Server is up and running",
	}

	return c.JSON(res)
}

// @description Returns a valid list of valid product keys.
// @description Any of these product keys can be used in the <PRODUCT> value of other endpoints. Please note many of these products are used for internal tools only and many have been EOL’d.
// @Param eol			query 	bool 	false 	"EOL Products"
// @Success 200 {object} omnitruck.ItemList
// @Failure 500 {object} services.ErrorResponse
// @Router /products [get]
func (server *TrialService) productsHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Eol: c.Query("eol", "false"),
	}

	var data omnitruck.ItemList
	request := server.Omnitruck.Products(params, &data)

	if params.Eol != "true" {
		data = omnitruck.FilterList(data, omnitruck.EolProductName)
	}

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}
}

// @description Returns a valid list of valid platform keys along with full friendly names.
// @description Any of these platform keys can be used in the p query string value in various endpoints below.
// @Success 200 {object} omnitruck.PlatformList
// @Failure 500 {object} services.ErrorResponse
// @Router /platforms [get]
func (server *TrialService) platformsHandler(c *fiber.Ctx) error {
	var data omnitruck.PlatformList
	request := server.Omnitruck.Platforms().ParseData(&data)

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}
}

// @description Returns a valid list of valid platform keys along with friendly names.
// @description Any of these architecture keys can be used in the p query string value in various endpoints below.
// @Success 200 {object} omnitruck.ItemList
// @Failure 500 {object} services.ErrorResponse
// @Router /architectures [get]
func (server *TrialService) architecturesHandler(c *fiber.Ctx) error {

	var data omnitruck.ItemList
	request := server.Omnitruck.Architectures().ParseData(&data)

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}
}

// @description Get the latest version number for a particular channel and product combination.
// @Param channel 		path 	string 	true 	"Channel" Enums(current, stable)
// @Param product   	path 	string 	true 	"Product"
// @Param license_id 	header 	string 	false 	"License ID"
// @Success 200 {object} omnitruck.ProductVersion
// @Failure 400 {object} services.ErrorResponse
// @Failure 403 {object} services.ErrorResponse
// @Router /{channel}/{product}/versions/latest [get]
func (server *TrialService) latestVersionHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Channel: c.Params("channel"),
		Product: c.Params("product"),
	}
	err, ok := server.ValidateRequest(params, c)
	if !ok {
		return err
	}

	var data omnitruck.ProductVersion
	request := server.Omnitruck.LatestVersion(params).ParseData(&data)

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}

}

// @description Get a list of all available version numbers for a particular channel and product combination
// @Param channel 		path 	string 	true 	"Channel" Enums(current, stable)
// @Param product   	path 	string 	true 	"Product"
// @Param license_id 	header 	string 	false 	"License ID"
// @Param eol			query 	bool 	false 	"EOL Products" Default(false)
// @Success 200 {object} omnitruck.ItemList
// @Failure 400 {object} services.ErrorResponse
// @Failure 403 {object} services.ErrorResponse
// @Router /{channel}/{product}/versions/all [get]
func (server *TrialService) productVersionsHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Channel: c.Params("channel"),
		Product: c.Params("product"),
		Eol:     c.Query("eol", "false"),
	}

	err, ok := server.ValidateRequest(params, c)
	if !ok {
		return err
	}

	var data []omnitruck.ProductVersion
	request := server.Omnitruck.ProductVersions(params).ParseData(&data)

	if params.Eol != "true" {
		data = omnitruck.FilterProductList(data, params.Product, omnitruck.EolProductVersion)
	}

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}

}

// @description Get the full list of all packages for a particular channel and product combination.
// @description By default all packages for the latest version are returned. If the v query string parameter is included the packages for the specified version are returned.
// @Param channel 		path 	string 	true 	"Channel" Enums(current, stable)
// @Param product   	path 	string 	true 	"Product" Example(chef)
// @Param v				query	string	false	"Version"
// @Param license_id 	header 	string 	false 	"License ID"
// @Param eol			query 	bool 	false 	"EOL Products" Default(false)
// @Success 200 {object} omnitruck.PackageList
// @Failure 400 {object} services.ErrorResponse
// @Failure 403 {object} services.ErrorResponse
// @Router /{channel}/{product}/packages [get]
func (server *TrialService) productPackagesHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Channel: c.Params("channel"),
		Product: c.Params("product"),
		Version: c.Query("v"),
		Eol:     c.Query("eol"),
	}

	err, ok := server.ValidateRequest(params, c)
	if !ok {
		return err
	}

	var data omnitruck.PackageList
	request := server.Omnitruck.ProductPackages(params).ParseData(&data)

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}

}

// @description Get details for a particular package.
// @description The `ACCEPT` HTTP header with a value of `application/json` must be provided in the request for a JSON response to be returned
// @Param channel 		path 	string 	true 	"Channel" 			Enums(current, stable)
// @Param product   	path 	string 	true 	"Product" 			Example(chef)
// @Param p				query	string	true	"Platform, valid values are returned from the `/platforms` endpoint." 				Example(ubuntu)
// @Param pv			query	string	true	"Platform Version, possible values depend on the platform. For example, Ubuntu: 16.04, or 18.04 or for macOS: 10.14 or 10.15." 	Example(20.04)
// @Param m				query	string	true	"Machine architecture, valid values are returned by the `/architectures` endpoint."	Example(x86_64)
// @Param v				query	string	false	"Version of the product to be installed. A version always takes the form `x.y.z`"			Default(latest)
// @Param license_id 	header 	string 	false 	"License ID"
// @Param eol			query 	bool 	false 	"EOL Products" Default(false)
// @Success 200 {object} omnitruck.PackageMetadata
// @Failure 400 {object} services.ErrorResponse
// @Failure 403 {object} services.ErrorResponse
// @Router /{channel}/{product}/metadata [get]
func (server *TrialService) productMetadataHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Channel:         c.Params("channel"),
		Product:         c.Params("product"),
		Version:         c.Query("v"),
		Platform:        c.Query("p"),
		PlatformVersion: c.Query("pv"),
		Architecture:    c.Query("m"),
	}

	err, ok := server.ValidateRequest(params, c)
	if !ok {
		return err
	}

	var data omnitruck.PackageMetadata
	request := server.Omnitruck.ProductMetadata(params).ParseData(&data)

	if request.Ok {
		return server.SendResponse(c, &data)
	} else {
		return server.SendError(c, request)
	}

}

// @description Get details for a particular package.
// @description The `ACCEPT` HTTP header with a value of `application/json` must be provided in the request for a JSON response to be returned
// @Param channel 		path 	string 	true 	"Channel" 			Enums(current, stable)
// @Param product   	path 	string 	true 	"Product" 			Example(chef)
// @Param p				query	string	true	"Platform, valid values are returned from the `/platforms` endpoint." 				Example(ubuntu)
// @Param pv			query	string	true	"Platform Version, possible values depend on the platform. For example, Ubuntu: 16.04, or 18.04 or for macOS: 10.14 or 10.15." 	Example(20.04)
// @Param m				query	string	true	"Machine architecture, valid values are returned by the `/architectures` endpoint."	Example(x86_64)
// @Param v				query	string	false	"Version of the product to be installed. A version always takes the form `x.y.z`"			Default(latest)
// @Param license_id 	header 	string 	false 	"License ID"
// @Param eol			query 	bool 	false 	"EOL Products" Default(false)
// @Success 302
// @Failure 400 {object} services.ErrorResponse
// @Failure 403 {object} services.ErrorResponse
// @Router /{channel}/{product}/download [get]
func (server *TrialService) productDownloadHandler(c *fiber.Ctx) error {
	params := &omnitruck.RequestParams{
		Channel:         c.Params("channel"),
		Product:         c.Params("product"),
		Version:         c.Query("v"),
		Platform:        c.Query("p"),
		PlatformVersion: c.Query("pv"),
		Architecture:    c.Query("m"),
	}

	err, ok := server.ValidateRequest(params, c)
	if !ok {
		return err
	}

	var data omnitruck.PackageMetadata
	request := server.Omnitruck.ProductDownload(params).ParseData(&data)

	if request.Ok {
		server.Log.Infof("Redirecting user to %s", data.Url)
		return c.Redirect(data.Url, 302)
	} else {
		return server.SendError(c, request)
	}
}

func (server *TrialService) buildRouter() {
	server.App.Get("/swagger/*", swagger.New(swagger.Config{
		InstanceName: "Trial",
	}))

	server.App.Get("/", server.HealthCheck)
	server.App.Get("/products", server.productsHandler)
	server.App.Get("/platforms", server.platformsHandler)
	server.App.Get("/architectures", server.architecturesHandler)
	server.App.Get("/:channel/:product/versions/latest", server.latestVersionHandler)
	server.App.Get("/:channel/:product/versions/all", server.productVersionsHandler)
	server.App.Get("/:channel/:product/packages", server.productPackagesHandler)
	server.App.Get("/:channel/:product/metadata", server.productMetadataHandler)
	server.App.Get("/:channel/:product/download", server.productDownloadHandler)

	server.App.Get("/status", server.HealthCheck)
}
