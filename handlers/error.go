package handlers

import "github.com/labstack/echo/v4"

func CustomHTTPErrorHandler(err error, c echo.Context) {
    code := 500
    if he, ok := err.(*echo.HTTPError); ok {
        code = he.Code
    }
    c.Logger().Error(err)
    if code >= 500 {
        c.JSON(code, "Something went wrong")
    } else {
        c.JSON(code, err)
    }
}

