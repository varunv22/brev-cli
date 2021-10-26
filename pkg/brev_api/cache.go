package brev_api

import "github.com/brevdev/brev-cli/pkg/files"

// Helper functions
func getOrgs() []Organization {
	client, _ := NewClient()
	orgs, _ := client.GetOrgs()

	return orgs
}

func getWorkspaces(orgID string) []Workspace {
	// orgID := getOrgID(orgName)

	client, _ := NewClient()
	workspaces, _ := client.GetWorkspaces(orgID)

	return workspaces
}


type CacheableWorkspace struct {
	OrgID string `json:"orgID`
	Workspaces []Workspace `json:"workspaces"`
}

// BANANA: this one just didn't work
// func write_individual_workspace_cache(orgID string, t *terminal.Terminal) error {
// 	var worspaceCache []CacheableWorkspace;
// 	path := files.GetWorkspacesCacheFilePath()
// 	err := files.ReadJSON(path, &worspaceCache)
// 	if err!=nil {
// 		return err
// 	}
// 	wss := getWorkspaces(orgID)
// 	for _, v := range worspaceCache {
// 		if v.OrgID == orgID {
// 			v.Workspaces = wss
// 			t.Vprintf("%d %s", len(v.Workspaces), v.OrgID)
// 			wsc := worspaceCache
// 			err := files.OverwriteJSON(path, wsc)
// 			if err!=nil {
// 				return err
// 			}
// 			return nil;
// 		}
// 	}
// 	return nil // BANANA: should this error cus it shouldn't get here???
// }

func Write_caches() error {

	orgs := getOrgs()
	path := files.GetOrgCacheFilePath()
	err := files.OverwriteJSON(path, orgs)
	if err!=nil {
		return err
	}

	var worspaceCache []CacheableWorkspace;
	for _, v := range orgs {
		wss := getWorkspaces(v.ID)
		worspaceCache = append(worspaceCache, CacheableWorkspace{
			OrgID: v.ID, Workspaces: wss,
		})
	}
	path_ws := files.GetWorkspacesCacheFilePath()
	err2 := files.OverwriteJSON(path_ws, worspaceCache)
	if err2!=nil {
		return err2
	}
	return nil
}

func Get_org_cache_data() ([]Organization,error) {
	path := files.GetOrgCacheFilePath()
	exists, err := files.Exists(path, false)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = Write_caches()
		if (err != nil) {
			return nil, err
		}
	}

	var orgCache []Organization
	err = files.ReadJSON(path, &orgCache)
	if err!=nil {
		return nil, err
	}
	return orgCache, nil
}

func Get_ws_cache_data() ([]CacheableWorkspace,error) {
	path := files.GetWorkspacesCacheFilePath()
	exists, err := files.Exists(path, false)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = Write_caches()
		if (err != nil) {
			return nil, err
		}
	}

	var wsCache []CacheableWorkspace
	err = files.ReadJSON(path, &wsCache)
	if err!=nil {
		return nil, err
	}
	return wsCache, nil
}

