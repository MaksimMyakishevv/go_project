package services

import (
	"new/models"
	"strings"
)

type GetAddressesService struct{}

func (s *GetAddressesService) GetAddresses(addresses models.Addresses) (string, error) {

	response := strings.Builder{}
	components := addresses.Components

	for _, item := range components {
		response.WriteString("Kind: " + item["kind"] + ", Name: " + item["name"] + ", ")
	}

    // Trim trailing comma and space if any
    result := response.String()
    if len(result) > 0 {
        result = result[:len(result)-2] // Remove last comma and space
    }

    return result, nil
}