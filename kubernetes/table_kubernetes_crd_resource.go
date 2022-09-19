package kubernetes

import (
	"context"
	"strings"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/turbot/steampipe-plugin-sdk/v3/plugin"
)

func tableKubernetesCRDResource(ctx context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "kubernetes_crd_resource",
		Description: "Cron jobs are useful for creating periodic and recurring tasks, like running backups or sending emails.",
		List: &plugin.ListConfig{
			ParentHydrate: listK8sCRDs,
			Hydrate:       listK8sCRDResources,
		},
		Columns: k8sCRDResourceCommonColumns([]*plugin.Column{}),
	}
}

type CRDResourceInfo struct {
	Kind        string
	APIVersion  string
	Name        string
	Namespace   string
	Annotations interface{}
	Spec        interface{}
}

//// HYDRATE FUNCTIONS

func listK8sCRDResources(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("listK8sCRDResources")

	version := h.Item.(v1.CustomResourceDefinition).Spec.Versions[0].Name
	groupName := h.Item.(v1.CustomResourceDefinition).Spec.Group
	object := h.Item.(v1.CustomResourceDefinition).Spec.Names.Plural

	clientset, err := GetNewClientDynamic(ctx, d)
	if err != nil {
		return nil, err
	}

	resourceId := schema.GroupVersionResource{
		Group:    groupName,
		Version:  version,
		Resource: object,
	}

	response, err := clientset.Resource(resourceId).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// panic(response.Items)
	for _, crd := range response.Items {
		ob := crd.Object
		var annotation interface{}
		annotation = strings.TrimLeft(strings.TrimRight(ob["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})["kubectl.kubernetes.io/last-applied-configuration"].(string), "\""), "\"")

		d.StreamListItem(ctx, &CRDResourceInfo{
			Kind:        ob["kind"].(string),
			APIVersion:  ob["apiVersion"].(string),
			Name:        ob["metadata"].(map[string]interface{})["name"].(string),
			Annotations: annotation,
			Namespace:   ob["metadata"].(map[string]interface{})["namespace"].(string),
			Spec:        ob["spec"],
		})

		// Context can be cancelled due to manual cancellation or the limit has been hit
		if d.QueryStatus.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}

	return nil, nil
}
