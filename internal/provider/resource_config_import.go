package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &configImportResource{}
var _ resource.ResourceWithConfigure = &configImportResource{}
var _ resource.ResourceWithImportState = &configImportResource{}

// configImportResource applies configuration exports.
type configImportResource struct {
	client *client.Client
}

type configImportModel struct {
	ID         types.String `tfsdk:"id"`
	ConfigJSON types.String `tfsdk:"config_json"`
}

// NewConfigImportResource constructs a new config import resource.
func NewConfigImportResource() resource.Resource {
	return &configImportResource{}
}

// Metadata sets the resource type name.
func (r *configImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_import"
}

// Schema defines the config import schema.
func (r *configImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies Open WebUI configuration keys from a JSON object. This is a singleton resource.\n\n" +
			"Open WebUI v0.11.0 stores configuration one key per row and merges an import into it, so `config_json` may hold as few keys as you want to manage. " +
			"Terraform tracks only the keys `config_json` names; every other key is left as the server has it, and never enters state.\n\n" +
			"Keys are flat and dotted, such as `ui.banners` and `audio.stt.engine`. " +
			"Open WebUI v0.9.x exported a nested tree instead, so a `config_json` captured from v0.9.x writes rows named `ui` and `code_execution` that nothing reads. The provider rejects that shape.\n\n" +
			"~> **Warning:** `config_json` may contain secrets. Treat the Terraform state for this resource as sensitive.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier of this singleton resource. Always `config_import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_json": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Open WebUI configuration keys to apply, as a JSON object of flat dotted keys. The `openwebui_config_export` data source produces the full set.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *configImportResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create applies the configuration import.
func (r *configImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	var plan configImportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConfigImport(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the managed keys from the configuration export.
func (r *configImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	var state configImportModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exported, err := r.client.ExportConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read config export failed", err.Error())
		return
	}

	managed := decodeOptionalJSON(state.ConfigJSON, path.Root("config_json"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	configJSON, diags := refreshedConfigJSON(state.ConfigJSON, managed, exported)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed := configImportModel{
		ID:         types.StringValue("config_import"),
		ConfigJSON: configJSON,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

// Update reapplies the configuration import.
func (r *configImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	var plan configImportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConfigImport(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *configImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *configImportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyConfigImport(ctx context.Context, apiClient *client.Client, plan configImportModel) (configImportModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := decodeOptionalJSON(plan.ConfigJSON, path.Root("config_json"), &diags)
	if diags.HasError() {
		return configImportModel{}, diags
	}
	if config == nil {
		diags.AddAttributeError(
			path.Root("config_json"),
			"Missing config JSON",
			"config_json must contain a JSON object describing the full configuration export.",
		)
		return configImportModel{}, diags
	}

	if nested := nestedConfigKeys(config); len(nested) > 0 {
		diags.AddAttributeError(
			path.Root("config_json"),
			"Nested configuration keys",
			fmt.Sprintf(
				"Open WebUI v0.11.0 stores configuration as flat dotted keys, such as ui.banners. These keys hold an object and carry no dot: %s. "+
					"A configuration exported from Open WebUI v0.9.x has this shape, and importing it writes rows that nothing reads. Export the configuration again from v0.11.0.",
				strings.Join(nested, ", "),
			),
		)
		return configImportModel{}, diags
	}

	// POST /configs/import merges the keys it is given and answers with the
	// whole configuration, every key of it. Recording that answer would replace
	// the value Terraform planned, so the plan's own text is the state value.
	if _, err := apiClient.ImportConfig(ctx, config); err != nil {
		diags.AddError("Import config failed", err.Error())
		return configImportModel{}, diags
	}

	state := configImportModel{
		ID:         types.StringValue("config_import"),
		ConfigJSON: plan.ConfigJSON,
	}

	return state, diags
}

// nestedConfigKeys names the top-level keys that hold an object and have no dot
// in them. Every key Open WebUI v0.11.0 defines is dotted except webhook_url,
// which holds a string, so an object under a bare key is a v0.9.x export.
func nestedConfigKeys(config map[string]any) []string {
	var nested []string
	for key, value := range config {
		if strings.Contains(key, ".") {
			continue
		}
		if _, isObject := value.(map[string]any); isObject {
			nested = append(nested, key)
		}
	}

	sort.Strings(nested)
	return nested
}

// refreshedConfigJSON reports the configuration as state should hold it: the
// managed keys only, and the practitioner's own text whenever the server agrees
// with it, so that a refresh does not rewrite the value over key order or
// whitespace alone. A state with no config_json at all, which is what an import
// produces, takes the whole export.
func refreshedConfigJSON(current types.String, managed, exported map[string]any) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	source := exported
	if !current.IsNull() && !current.IsUnknown() {
		source = make(map[string]any, len(managed))
		for key := range managed {
			if value, ok := exported[key]; ok {
				source[key] = value
			}
		}

		if reflect.DeepEqual(source, managed) {
			return current, diags
		}
	}

	encoded, err := encodeOptionalJSON(source)
	if err != nil {
		diags.AddError("Serialize config export", err.Error())
	}

	return encoded, diags
}
