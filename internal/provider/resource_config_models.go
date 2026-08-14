package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &modelsConfigResource{}
var _ resource.ResourceWithConfigure = &modelsConfigResource{}
var _ resource.ResourceWithImportState = &modelsConfigResource{}

// modelsConfigResource manages default model configuration.
type modelsConfigResource struct {
	client *client.Client
}

type modelsConfigModel struct {
	ID                       types.String `tfsdk:"id"`
	DefaultModels            types.String `tfsdk:"default_models"`
	DefaultPinnedModels      types.String `tfsdk:"default_pinned_models"`
	ModelOrderList           types.List   `tfsdk:"model_order_list"`
	DefaultModelMetadataJSON types.String `tfsdk:"default_model_metadata_json"`
	DefaultModelParamsJSON   types.String `tfsdk:"default_model_params_json"`
}

// NewModelsConfigResource constructs a new models config resource.
func NewModelsConfigResource() resource.Resource {
	return &modelsConfigResource{}
}

// Metadata sets the resource type name.
func (r *modelsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_models_config"
}

// Schema defines the models config schema.
func (r *modelsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the model list every user starts from: which models a new chat opens with, which ones sit pinned at the top, the order the picker shows them in, and the default metadata and parameters.\n\n`POST /api/v1/configs/models` writes all five keys of the form it receives, so an attribute this configuration leaves out is written as null and the stored value is lost. The two JSON attributes are the exception: the provider reads them back and carries them across a write that does not name them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of this singleton resource. Always `models`.",
				MarkdownDescription: "Identifier of this singleton resource. Always `models`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_models": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Comma-separated model IDs a new chat opens with. Stored as `DEFAULT_MODELS`.",
				MarkdownDescription: "Comma-separated model IDs a new chat opens with. Stored as `DEFAULT_MODELS`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_pinned_models": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Comma-separated model IDs pinned to the top of the model picker. Stored as `DEFAULT_PINNED_MODELS`.",
				MarkdownDescription: "Comma-separated model IDs pinned to the top of the model picker. Stored as `DEFAULT_PINNED_MODELS`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"model_order_list": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "Model IDs in the order the picker lists them. Stored as `MODEL_ORDER_LIST`.",
				MarkdownDescription: "Model IDs in the order the picker lists them. Stored as `MODEL_ORDER_LIST`.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"default_model_metadata_json": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Default model metadata as a JSON object. Open WebUI stores it under models.default_metadata.",
				MarkdownDescription: "Default model metadata as a JSON object. Open WebUI stores it under `models.default_metadata`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_model_params_json": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Default model parameters as a JSON object. Open WebUI stores it under models.default_params.",
				MarkdownDescription: "Default model parameters as a JSON object. Open WebUI stores it under `models.default_params`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *modelsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the models config.
func (r *modelsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing models config.")
		return
	}

	var plan modelsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyModelsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the models config.
func (r *modelsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing models config.")
		return
	}

	var recorded modelsConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetModelsConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read models config failed", err.Error())
		return
	}

	orderList, listDiags := flattenStringSlice(ctx, config.ModelOrderList)
	resp.Diagnostics.Append(listDiags...)

	metadataJSON, metadataDiags := jsonRefreshValue(recorded.DefaultModelMetadataJSON, config.DefaultModelMetadata, "default_model_metadata_json")
	resp.Diagnostics.Append(metadataDiags...)

	paramsJSON, paramsDiags := jsonRefreshValue(recorded.DefaultModelParamsJSON, config.DefaultModelParams, "default_model_params_json")
	resp.Diagnostics.Append(paramsDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state := modelsConfigModel{
		ID:                       types.StringValue("models"),
		DefaultModels:            stringValueOrNull(config.DefaultModels),
		DefaultPinnedModels:      stringValueOrNull(config.DefaultPinnedModels),
		ModelOrderList:           orderList,
		DefaultModelMetadataJSON: metadataJSON,
		DefaultModelParamsJSON:   paramsJSON,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new models config.
func (r *modelsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing models config.")
		return
	}

	var plan modelsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyModelsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *modelsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing models config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *modelsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyModelsConfig(ctx context.Context, apiClient *client.Client, plan modelsConfigModel) (modelsConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var orderList []string
	if !plan.ModelOrderList.IsNull() && !plan.ModelOrderList.IsUnknown() {
		orderList = expandStringList(ctx, plan.ModelOrderList, path.Root("model_order_list"), &diags)
	}

	metadata := decodeOptionalJSON(plan.DefaultModelMetadataJSON, path.Root("default_model_metadata_json"), &diags)
	params := decodeOptionalJSON(plan.DefaultModelParamsJSON, path.Root("default_model_params_json"), &diags)
	if diags.HasError() {
		return modelsConfigModel{}, diags
	}

	// POST /configs/models writes every field of the form it receives, so a
	// field the plan leaves unknown has to carry the server's current value.
	// Sending it as null erases the stored metadata and parameters.
	if plan.DefaultModelMetadataJSON.IsUnknown() || plan.DefaultModelParamsJSON.IsUnknown() {
		current, err := apiClient.GetModelsConfig(ctx)
		if err != nil {
			diags.AddError("Read models config failed", err.Error())
			return modelsConfigModel{}, diags
		}
		if plan.DefaultModelMetadataJSON.IsUnknown() {
			metadata = current.DefaultModelMetadata
		}
		if plan.DefaultModelParamsJSON.IsUnknown() {
			params = current.DefaultModelParams
		}
	}

	form := client.ModelsConfigForm{
		DefaultModels:        stringPtr(plan.DefaultModels),
		DefaultPinnedModels:  stringPtr(plan.DefaultPinnedModels),
		ModelOrderList:       orderList,
		DefaultModelMetadata: metadata,
		DefaultModelParams:   params,
	}

	updated, err := apiClient.SetModelsConfig(ctx, form)
	if err != nil {
		diags.AddError("Update models config failed", err.Error())
		return modelsConfigModel{}, diags
	}

	order, listDiags := flattenStringSlice(ctx, updated.ModelOrderList)
	diags.Append(listDiags...)

	metadataJSON, encodeDiags := jsonStateValue(plan.DefaultModelMetadataJSON, updated.DefaultModelMetadata, "default_model_metadata_json")
	diags.Append(encodeDiags...)

	paramsJSON, encodeDiags := jsonStateValue(plan.DefaultModelParamsJSON, updated.DefaultModelParams, "default_model_params_json")
	diags.Append(encodeDiags...)

	if diags.HasError() {
		return modelsConfigModel{}, diags
	}

	state := modelsConfigModel{
		ID:                       types.StringValue("models"),
		DefaultModels:            stringValueOrNull(updated.DefaultModels),
		DefaultPinnedModels:      stringValueOrNull(updated.DefaultPinnedModels),
		ModelOrderList:           order,
		DefaultModelMetadataJSON: metadataJSON,
		DefaultModelParamsJSON:   paramsJSON,
	}

	return state, diags
}

// jsonStateValue keeps the configured JSON text when the plan carries one, so
// that re-encoding the server's answer cannot reorder keys or change whitespace
// and make the applied value differ from the planned one.
func jsonStateValue(planned types.String, serverValue map[string]any, attribute string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !planned.IsNull() && !planned.IsUnknown() {
		return planned, diags
	}

	encoded, err := encodeOptionalJSON(serverValue)
	if err != nil {
		diags.AddError(fmt.Sprintf("Serialize %s failed", attribute), err.Error())
	}

	return encoded, diags
}

// jsonRefreshValue keeps the recorded JSON text while it still describes the
// server's value, so a refresh reports real drift rather than a difference in
// key order or whitespace.
func jsonRefreshValue(recorded types.String, serverValue map[string]any, attribute string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !recorded.IsNull() && !recorded.IsUnknown() {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(recorded.ValueString()), &decoded); err == nil && reflect.DeepEqual(decoded, serverValue) {
			return recorded, diags
		}
	}

	encoded, err := encodeOptionalJSON(serverValue)
	if err != nil {
		diags.AddError(fmt.Sprintf("Serialize %s failed", attribute), err.Error())
	}

	return encoded, diags
}
