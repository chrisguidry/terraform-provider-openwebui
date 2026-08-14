package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// Conversions shared by the four engine config resources: RAG embedding, RAG,
// images, and audio.
//
// The three routers behind them write every key of the form they receive, so
// each resource sends a whole form on every write. Each engineX function
// therefore takes the planned value and the value Open WebUI currently holds,
// and picks the current one when the plan carries nothing. A field the
// configuration never names keeps what the server already stored instead of
// being replaced by a Pydantic default.

// engineString picks the planned string, or the stored one when the plan carries none.
func engineString(planned types.String, current *string) *string {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := planned.ValueString()

	return &value
}

// engineBool picks the planned boolean, or the stored one when the plan carries none.
func engineBool(planned types.Bool, current *bool) *bool {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := planned.ValueBool()

	return &value
}

// engineInt64 picks the planned integer, or the stored one when the plan carries none.
func engineInt64(planned types.Int64, current *int64) *int64 {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := planned.ValueInt64()

	return &value
}

// engineNumericInt picks the planned integer, or the stored one when the plan carries none.
func engineNumericInt(planned types.Int64, current *client.NumericInt) *client.NumericInt {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := client.NumericInt(planned.ValueInt64())

	return &value
}

// engineFloat64 picks the planned number, or the stored one when the plan carries none.
func engineFloat64(planned types.Float64, current *float64) *float64 {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := planned.ValueFloat64()

	return &value
}

// engineNumericString picks the planned value, or the stored one when the plan
// carries none. The empty string is a value here, not an absence: it is what
// clears a file limit.
func engineNumericString(planned types.String, current *client.NumericString) *client.NumericString {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	value := client.NumericString(planned.ValueString())

	return &value
}

// engineStringList picks the planned list, or the stored one when the plan
// carries none. The result is a pointer so that an omitted list reaches the API
// as an absent key rather than as a null, which several of these forms reject.
func engineStringList(ctx context.Context, planned types.List, current *[]string, attribute path.Path, diags *diag.Diagnostics) *[]string {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	values := expandStringList(ctx, planned, attribute, diags)
	if values == nil {
		values = []string{}
	}

	return &values
}

// engineStringListValue records a list the API returned, and null when it returned none.
func engineStringListValue(ctx context.Context, values *[]string, diags *diag.Diagnostics) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}

	list, d := types.ListValueFrom(ctx, types.StringType, *values)
	diags.Append(d...)

	return list
}

// engineJSON decodes a JSON attribute, or keeps the stored value when the plan
// carries none. The value may be an object or an array, so it decodes into any.
func engineJSON(planned types.String, current any, attribute path.Path, diags *diag.Diagnostics) any {
	if planned.IsNull() || planned.IsUnknown() {
		return current
	}

	raw := strings.TrimSpace(planned.ValueString())
	if raw == "" {
		return current
	}

	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		diags.AddAttributeError(
			attribute,
			"Invalid JSON value",
			fmt.Sprintf("Expected attribute %s to contain valid JSON text: %v", attribute.String(), err),
		)

		return current
	}

	return decoded
}

// engineJSONValue keeps the recorded text while it still describes the server's
// value, so a difference in key order or whitespace does not read as drift.
func engineJSONValue(recorded types.String, serverValue any, attribute string, diags *diag.Diagnostics) types.String {
	if !recorded.IsNull() && !recorded.IsUnknown() {
		var decoded any
		if err := json.Unmarshal([]byte(recorded.ValueString()), &decoded); err == nil && reflect.DeepEqual(decoded, serverValue) {
			return recorded
		}
	}

	encoded, err := encodeOptionalJSONValue(serverValue)
	if err != nil {
		diags.AddError(fmt.Sprintf("Serialize %s failed", attribute), err.Error())
	}

	return encoded
}

// engineBoolValue records a boolean the API returned, and null when it returned none.
func engineBoolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

// engineFloat64Value records a number the API returned, and null when it returned none.
func engineFloat64Value(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}

	return types.Float64Value(*value)
}

// engineNumericIntValue records an integer the API returned, and null when it returned none.
func engineNumericIntValue(value *client.NumericInt) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*value))
}

// engineNumericStringValue records a file limit the API returned. Open WebUI
// stores null for a limit the empty string cleared, so a recorded empty string
// survives a null reply and the configuration stops asking to clear it again.
func engineNumericStringValue(recorded types.String, value *client.NumericString) types.String {
	if value == nil {
		if !recorded.IsNull() && !recorded.IsUnknown() && recorded.ValueString() == "" {
			return recorded
		}

		return types.StringNull()
	}

	return types.StringValue(string(*value))
}

// engineTrimmedURL drops the slashes Open WebUI strips before it stores a
// ComfyUI base URL, so a configuration written with a trailing slash converges.
func engineTrimmedURL(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.Trim(*value, "/")

	return &trimmed
}

// engineTrimmedURLValue keeps the recorded URL while it names the same server
// the API returned. Terraform requires the applied value to equal the planned
// one, and the plan carries whatever the configuration wrote, so a state that
// took the server's trimmed form back would fail every apply that ends the URL
// in a slash.
func engineTrimmedURLValue(recorded types.String, value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	if !recorded.IsNull() && !recorded.IsUnknown() && strings.Trim(recorded.ValueString(), "/") == *value {
		return recorded
	}

	return types.StringValue(*value)
}
