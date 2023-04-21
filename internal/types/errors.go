/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// JSONFailureResponse describes a failure message returned by the API
type JSONFailureResponse struct {
	Message string `json:"error" example:"err"`
}

// JSONHealthResponse describes a message returned by the API health endpoint
type JSONHealthResponse struct {
	Status string `json:"status" example:"healthy"`
}

// JSONSuccessResponse describes a success message returned by the API
type JSONSuccessResponse struct {
	Message string `json:"message" example:"success"`
}
