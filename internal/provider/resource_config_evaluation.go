package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &evaluationConfigResource{}
var _ resource.ResourceWithConfigure = &evaluationConfigResource{}
var _ resource.ResourceWithImportState = &evaluationConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewEvaluationConfigResource)
}

// evaluationConfigResource manages the arena evaluation settings.
type evaluationConfigResource struct {
	client *client.Client
}

type evaluationConfigModel struct {
	ID                          types.String `tfsdk:"id"`
	EnableEvaluationArenaModels types.Bool   `tfsdk:"enable_evaluation_arena_models"`
	EvaluationArenaModelsJSON   types.String `tfsdk:"evaluation_arena_models_json"`
}

// NewEvaluationConfigResource constructs a new evaluation config resource.
func NewEvaluationConfigResource() resource.Resource {
	return &evaluationConfigResource{}
}

// Metadata sets the resource type name.
func (r *evaluationConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_evaluation_config"
}

// Schema defines the evaluation config schema.
func (r *evaluationConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the arena evaluation settings of Open WebUI. An arena model puts two models against each other and collects the user's preference.\n\n" +
			"An empty list is not the same as disabled: with `enable_evaluation_arena_models` true and no models listed, Open WebUI serves a built-in arena model at request time. That model is never written back to the configuration, so Terraform reports no drift for it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of this singleton resource. Always `evaluation`.",
				MarkdownDescription: "Identifier of this singleton resource. Always `evaluation`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_evaluation_arena_models": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether arena models are offered in the model list.",
				MarkdownDescription: "Whether arena models are offered in the model list.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"evaluation_arena_models_json": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Arena models as a JSON array of objects. Open WebUI validates nothing about the elements, so the value is carried as text. " +
					"Each element needs an id, a name, and a meta object, or model listing fails at request time.",
				MarkdownDescription: "Arena models as a JSON array of objects. Open WebUI validates nothing about the elements, so the value is carried as text. " +
					"Each element needs an `id`, a `name`, and a `meta` object, or model listing fails at request time.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *evaluationConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create writes the evaluation config.
func (r *evaluationConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation config.")
		return
	}

	var plan evaluationConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyEvaluationConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the evaluation config.
func (r *evaluationConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation config.")
		return
	}

	var recorded evaluationConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetEvaluationConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read evaluation config failed", err.Error())
		return
	}

	modelsJSON, diags := jsonListRefreshValue(recorded.EvaluationArenaModelsJSON, config.ArenaModels, "evaluation_arena_models_json")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := evaluationConfigModel{
		ID:                          types.StringValue("evaluation"),
		EnableEvaluationArenaModels: types.BoolValue(config.EnableArenaModels),
		EvaluationArenaModelsJSON:   modelsJSON,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the evaluation config.
func (r *evaluationConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation config.")
		return
	}

	var plan evaluationConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyEvaluationConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *evaluationConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *evaluationConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyEvaluationConfig writes the plan. The route applies each field only when
// it is present, so an attribute the plan leaves unknown is left out of the
// request and keeps the value the instance holds.
func applyEvaluationConfig(ctx context.Context, apiClient *client.Client, plan evaluationConfigModel) (evaluationConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.EvaluationConfigForm{}

	if !plan.EnableEvaluationArenaModels.IsNull() && !plan.EnableEvaluationArenaModels.IsUnknown() {
		enabled := plan.EnableEvaluationArenaModels.ValueBool()
		form.EnableArenaModels = &enabled
	}

	if !plan.EvaluationArenaModelsJSON.IsNull() && !plan.EvaluationArenaModelsJSON.IsUnknown() {
		models := decodeJSONList(plan.EvaluationArenaModelsJSON, path.Root("evaluation_arena_models_json"), &diags)
		if diags.HasError() {
			return evaluationConfigModel{}, diags
		}
		form.ArenaModels = &models
	}

	updated, err := apiClient.SetEvaluationConfig(ctx, form)
	if err != nil {
		diags.AddError("Update evaluation config failed", err.Error())
		return evaluationConfigModel{}, diags
	}

	modelsJSON, encodeDiags := jsonListStateValue(plan.EvaluationArenaModelsJSON, updated.ArenaModels, "evaluation_arena_models_json")
	diags.Append(encodeDiags...)
	if diags.HasError() {
		return evaluationConfigModel{}, diags
	}

	state := evaluationConfigModel{
		ID:                          types.StringValue("evaluation"),
		EnableEvaluationArenaModels: types.BoolValue(updated.EnableArenaModels),
		EvaluationArenaModelsJSON:   modelsJSON,
	}

	return state, diags
}

// decodeJSONList maps a Terraform string attribute holding a JSON array into a
// Go slice.
func decodeJSONList(value types.String, attribute path.Path, diags *diag.Diagnostics) []any {
	raw := strings.TrimSpace(value.ValueString())
	if raw == "" {
		return []any{}
	}

	var result []any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		diags.AddAttributeError(
			attribute,
			"Invalid JSON value",
			fmt.Sprintf("Expected attribute %s to contain a JSON array: %v", attribute.String(), err),
		)
		return nil
	}

	return result
}

// encodeJSONList converts a slice into JSON text. A nil slice becomes null,
// which is how an unset list reads back.
func encodeJSONList(values []any) (types.String, error) {
	if values == nil {
		return types.StringNull(), nil
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return types.StringNull(), fmt.Errorf("marshal JSON: %w", err)
	}

	return types.StringValue(string(encoded)), nil
}

// jsonListStateValue keeps the configured JSON text when the plan carries one,
// so that re-encoding the server's answer cannot change whitespace and make the
// applied value differ from the planned one.
func jsonListStateValue(planned types.String, serverValue []any, attribute string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !planned.IsNull() && !planned.IsUnknown() {
		return planned, diags
	}

	encoded, err := encodeJSONList(serverValue)
	if err != nil {
		diags.AddError(fmt.Sprintf("Serialize %s failed", attribute), err.Error())
	}

	return encoded, diags
}

// jsonListRefreshValue keeps the recorded JSON text while it still describes the
// server's value, so a refresh reports real drift rather than a difference in
// whitespace.
func jsonListRefreshValue(recorded types.String, serverValue []any, attribute string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !recorded.IsNull() && !recorded.IsUnknown() {
		var decoded []any
		if err := json.Unmarshal([]byte(recorded.ValueString()), &decoded); err == nil && reflect.DeepEqual(decoded, serverValue) {
			return recorded, diags
		}
	}

	encoded, err := encodeJSONList(serverValue)
	if err != nil {
		diags.AddError(fmt.Sprintf("Serialize %s failed", attribute), err.Error())
	}

	return encoded, diags
}
