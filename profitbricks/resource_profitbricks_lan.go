package profitbricks

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func resourceProfitBricksLan() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksLanCreate,
		Read:   resourceProfitBricksLanRead,
		Update: resourceProfitBricksLanUpdate,
		Delete: resourceProfitBricksLanDelete,
		Schema: map[string]*schema.Schema{

			"public": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksLanCreate(d *schema.ResourceData, meta interface{}) error {
	request := profitbricks.CreateLanRequest{
		Properties: profitbricks.CreateLanProperties{
			Public: d.Get("public").(bool),
		},
	}

	log.Printf("[DEBUG] NAME %s", d.Get("name"))
	if d.Get("name") != nil {
		request.Properties.Name = d.Get("name").(string)
	}

	lan := profitbricks.CreateLan(d.Get("datacenter_id").(string), request)

	log.Printf("[DEBUG] LAN ID: %s", lan.Id)
	log.Printf("[DEBUG] LAN RESPONSE: %s", lan.Response)

	if lan.StatusCode > 299 {
		return fmt.Errorf("An error occured while creating a lan: %s", lan.Response)
	}

	err := waitTillProvisioned(meta, lan.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(lan.Id)
	return resourceProfitBricksLanRead(d, meta)
}

func resourceProfitBricksLanRead(d *schema.ResourceData, meta interface{}) error {
	lan := profitbricks.GetLan(d.Get("datacenter_id").(string), d.Id())

	if lan.StatusCode > 299 {
		if lan.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("An error occured while fetching a lan ID %s %s", d.Id(), lan.Response)
	}

	d.Set("public", lan.Properties.Public)
	d.Set("name", lan.Properties.Name)
	d.Set("ip_failover", lan.Properties.IpFailover)
	d.Set("datacenter_id", d.Get("datacenter_id").(string))
	return nil
}

func resourceProfitBricksLanUpdate(d *schema.ResourceData, meta interface{}) error {
	properties := &profitbricks.LanProperties{}
	newValue := d.Get("public")
	properties.Public = newValue.(bool)
	if d.HasChange("name") {
		_, newValue := d.GetChange("name")
		properties.Name = newValue.(string)
	}

	if properties != nil {
		lan := profitbricks.PatchLan(d.Get("datacenter_id").(string), d.Id(), *properties)
		if lan.StatusCode > 299 {
			return fmt.Errorf("An error occured while patching a lan ID %s %s", d.Id(), lan.Response)
		}
		err := waitTillProvisioned(meta, lan.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}
	return resourceProfitBricksLanRead(d, meta)
}

func resourceProfitBricksLanDelete(d *schema.ResourceData, meta interface{}) error {
	dcId := d.Get("datacenter_id").(string)
	resp := profitbricks.DeleteLan(dcId, d.Id())
	if resp.StatusCode > 299 {
		//try again in 120 seconds
		time.Sleep(120 * time.Second)
		resp = profitbricks.DeleteLan(dcId, d.Id())
		if resp.StatusCode > 299 && resp.StatusCode != 404 {
			return fmt.Errorf("An error occured while deleting a lan dcId %s ID %s %s", d.Get("datacenter_id").(string), d.Id(), string(resp.Body))
		}
	}

	if resp.Headers.Get("Location") != "" {
		err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}
	d.SetId("")
	return nil
}
