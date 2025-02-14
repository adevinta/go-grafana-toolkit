package client

import (
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
)

// DashboardClient defines operations for uploading and updating dashboards
// in a Grafana instance.
type DashboardClient interface {
	// UploadDashboard creates or updates a dashboard in Grafana.
	UploadDashboard(dashboard *Dashboard) error

	// GetDashboard retrieves a dashboard by its UID.
	GetDashboard(uid string) (*Dashboard, error)

	// DeleteDashboard removes a dashboard identified by its UID.
	DeleteDashboard(uid string) error

	// EnsureFolder creates a folder if it doesn't exist or returns existing folder.
	EnsureFolder(folder string) (*Folder, error)

	// GetDataSource retrieves a datasource by its name.
	GetDataSource(name string) (*Datasource, error)
}

type JSON interface{}

// Folder represents a Grafana folder with its UID and title
type Folder struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
}

// Dashboard represents a Grafana dashboard with its metadata and content
type Dashboard struct {
	UID       string `json:"uid"`
	Title     string `json:"title"`
	FolderUID string
	Dashboard JSON
	Meta      *models.DashboardMeta
}

type Datasource = models.DataSource

func (sc *StackClient) GetDataSource(name string) (*Datasource, error) {

	res, err := sc.httpApi.Datasources.GetDataSourceByName(name)

	if err != nil {
		return nil, fmt.Errorf("failed to get datasource for %s: %w", name, err)
	}

	if res.Payload == nil {
		return nil, fmt.Errorf("received no datasource data for: %s", name)
	}

	return res.Payload, nil
}

func (sc *StackClient) DeleteDashboard(uid string) error {

	_, err := sc.httpApi.Dashboards.DeleteDashboardByUID(uid)

	if err != nil {
		return fmt.Errorf("failed to delete dashboard %s: %w", uid, err)
	}

	return nil
}

func (sc *StackClient) UploadDashboard(dashboard *Dashboard) error {

	saveDashboardCmd := &models.SaveDashboardCommand{
		Dashboard: dashboard.Dashboard,
		FolderUID: dashboard.FolderUID,
		Overwrite: true,
		IsFolder:  false,
		Message:   "toolkit/grafana automated dashboard upload",
	}

	_, err := sc.httpApi.Dashboards.PostDashboard(saveDashboardCmd)
	if err != nil {
		return fmt.Errorf("failed to updload dashboard %s: %w", dashboard.UID, err)
	}

	return nil
}

func (sc *StackClient) GetDashboard(uid string) (*Dashboard, error) {

	res, err := sc.httpApi.Dashboards.GetDashboardByUID(uid)

	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard %s: %w", uid, err)
	}

	if res.Payload == nil || res.Payload.Dashboard == nil {
		return nil, fmt.Errorf("received no dashboard data for uid: %s", uid)
	}

	return &Dashboard{
		UID:       uid,
		Dashboard: res.Payload.Dashboard,
		Meta:      res.Payload.Meta,
	}, nil
}

func (sc *StackClient) GetFolder(folderName string) (*Folder, error) {
	foldersRes, err := sc.httpApi.Folders.GetFolders(folders.NewGetFoldersParams())

	if err != nil {
		return nil, fmt.Errorf("failed to get folders for  %s: %w", folderName, err)
	}

	for _, f := range foldersRes.Payload {
		if f.Title == folderName {
			return &Folder{
				UID:   f.UID,
				Title: f.Title,
			}, nil
		}
	}

	return nil, nil
}

func (sc *StackClient) EnsureFolder(folderName string) (*Folder, error) {

	folder, err := sc.GetFolder(folderName)

	if err != nil {
		return nil, fmt.Errorf("failed to get folders for %s: %w", folderName, err)
	}

	if folder != nil {
		return folder, nil
	}

	createFolderCmd := &models.CreateFolderCommand{Title: folderName}
	createRes, err := sc.httpApi.Folders.CreateFolder(createFolderCmd)

	if err != nil {
		return nil, fmt.Errorf("failed to create folder %s: %w", folderName, err)
	}

	return &Folder{
		UID:   createRes.Payload.UID,
		Title: createRes.Payload.Title,
	}, nil
}
