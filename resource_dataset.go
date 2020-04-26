package main

import (
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform/helper/schema"
)

type Dataset struct {
	ID          *uuid.UUID `json:"uuid,omitempty"`
	Name        string     `json:"name"`
	Compression string     `json:"compression,omitempty"`
	Quota       string     `json:"quota,omitempty"`
}

func (m *Dataset) LoadFromSchema(d *schema.ResourceData) error {

	if iid, ok := d.GetOk("uuid"); ok {
		uuid, _ := uuid.Parse(iid.(string))
		m.ID = &uuid
	}

	m.Name = d.Get("name").(string)
	m.Compression = d.Get("compression").(string)
	m.Quota = d.Get("quota").(string)

	return nil
}

func (m *Dataset) SaveToSchema(d *schema.ResourceData) error {
	d.Set("uuid", m.ID.String())
	d.Set("name", m.Name)
	d.Set("compression", m.Compression)
	d.Set("quota", m.Quota)

	return nil
}

func resourceDataset() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatasetCreate,
		Read:   resourceDatasetRead,
		Update: resourceDatasetUpdate,
		Delete: resourceDatasetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"compression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"quota": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDatasetCreate(d *schema.ResourceData, m interface{}) error {
	d.SetId("")

	client := m.(*IllumosClient)
	dataset := Dataset{}
	err := dataset.LoadFromSchema(d)
	if err != nil {
		return err
	}

	uuid, err := client.CreateDataset(&dataset)
	if err != nil {
		return err
	}

	d.SetId(uuid.String())

	err = resourceDatasetRead(d, m)
	return err
}

func resourceDatasetRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*IllumosClient)
	uuid, err := uuid.Parse(d.Id())
	if err != nil {
		log.Printf("Failed to parse incoming ID: %s", err)
		return err
	}

	dataset, err := client.GetDataset(uuid)
	if err != nil {
		log.Printf("Failed to retrieve dataset with ID %s.  Error: %s", d.Id(), err)
		return err
	}

	err = dataset.SaveToSchema(d)
	return err
}

func resourceDatasetUpdate(d *schema.ResourceData, m interface{}) error {
	datasetID, err := uuid.Parse(d.Id())
	if err != nil {
		return err
	}

	d.Partial(true)

	datasetUpdate := Dataset{
		ID: &datasetID,
	}
	datasetUpdate.Name = d.Get("name").(string)

	var properties []string

	if d.HasChange("compression") && !d.IsNewResource() {
		_, newValue := d.GetChange("compression")

		datasetUpdate.Compression = newValue.(string)
		properties = append(properties, "compression=\""+newValue.(string)+"\"")
	}

	if d.HasChange("quota") && !d.IsNewResource() {
		_, newValue := d.GetChange("quota")

		datasetUpdate.Compression = newValue.(string)
		properties = append(properties, "quota=\""+newValue.(string)+"\"")
	}

	if len(properties) > 0 {
		client := m.(*IllumosClient)

		err = client.UpdateDataset(&datasetUpdate, properties)
		if err != nil {
			return err
		}
	}

	d.Partial(false)
	err = resourceDatasetRead(d, m)
	return err
}

func resourceDatasetDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("Request to delete dataset with ID: %s\n", d.Id())

	client := m.(*IllumosClient)

	return client.DeleteDataset(d.Get("name").(string))
}
