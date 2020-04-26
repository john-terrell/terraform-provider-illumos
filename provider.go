package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema:       providerSchema(),
		ResourcesMap: providerResources(),
		//		DataSourcesMap: providerDataSources(),
		ConfigureFunc: providerConfigure,
	}
}

func providerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"host": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "Host address of the Illumos global zone.",
		},
		"user": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "User to authenticate with.",
		},
	}
}

func providerResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"illumos_dataset": resourceDataset(),
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client := IllumosClient{
		host: d.Get("host").(string),
		user: d.Get("user").(string),
	}

	return &client, nil
}
