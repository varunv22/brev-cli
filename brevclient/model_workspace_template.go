/*
 * Brev API
 *
 * Brev REST API.
 *
 * API version: 0.1.0
 * Contact: support@brev.dev
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package swagger

type WorkspaceTemplate struct {
	Id string `json:"id,omitempty"`
	Image string `json:"image,omitempty"`
	Name string `json:"name,omitempty"`
	Port int32 `json:"port,omitempty"`
	Public bool `json:"public,omitempty"`
	RegistryUri string `json:"registryUri,omitempty"`
	Type_ string `json:"type,omitempty"`
}
