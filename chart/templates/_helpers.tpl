{{/*
Expand the name of the chart.
*/}}
{{- define "ok8s.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ok8s.fullname" -}}
{{- if eq .Release.Namespace "default" }}
{{- fail "Release namespace cannot be 'default'. Install with -n <namespace>, e.g.: helm install ok8s ./chart -n opencode" }}
{{- end }}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if or (contains $name .Release.Name) (contains .Release.Name $name) }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ok8s.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ok8s.labels" -}}
helm.sh/chart: {{ include "ok8s.chart" . }}
{{ include "ok8s.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ok8s.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ok8s.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Generate opencode.jsonc config
*/}}
{{- define "ok8s.opencodeConfig" -}}
{
  "$schema": "https://opencode.ai/config.json",
  {{- if .Values.plugins.enabled }}
  "plugin": [
    "oh-my-opencode@latest",
    "@tarquinen/opencode-dcp@latest"{{- range .Values.plugins.npm }},
    "{{ . }}"{{- end }}
  ],
  {{- end }}
  "mcp": {
    {{- $first := true }}
    {{- range .Values.mcp.remote }}
    {{- if not $first }},{{ end }}
    "{{ .name }}": {
      "type": "remote",
      "url": "{{ .url }}"{{ if .enabled }},
      "enabled": true{{ end }}{{ if .headers }},
      "headers": {{ .headers | toJson }}{{ end }}{{ if .oauth }},
      "oauth": {{ .oauth | toJson }}{{ end }}
    }
    {{- $first = false }}
    {{- end }}
    {{- range .Values.mcp.laptopServers }}
    {{- if not $first }},{{ end }}
    "laptop_{{ .name }}": {
      "type": "remote",
      "url": "http://{{ $.Release.Name }}-egress-{{ .name }}:{{ .port }}"{{ if .enabled }},
      "enabled": true{{ end }}
    }
    {{- $first = false }}
    {{- end }}
  }{{- if or (.Values.skills.npm | len) (.Values.skills.config | len) }},
  "skills": {
    {{- if .Values.skills.npm }}"npm": {{ .Values.skills.npm | toJson }}{{- else }}"npm": []{{- end }},
    {{- if .Values.skills.config }}"config": {{ .Values.skills.config | toJson }}{{- else }}"config": []{{- end }}
  }
  {{- end }}
}
{{- end }}

{{/*
Get user count
*/}}
{{- define "ok8s.userCount" -}}
{{- len .Values.users }}
{{- end }}

{{/*
Resolve user name by pod ordinal index.
Uses a shell expression rendered into the template — the actual resolution
happens at runtime via the entrypoint script.
For the USER_NAME env var we render a comma-separated list and let the
entrypoint pick by index.
*/}}
{{- define "ok8s.userNameByIndex" -}}
{{- $names := list }}
{{- range .Values.users }}
{{- $names = append $names .name }}
{{- end }}
{{- join "," $names }}
{{- end }}

{{/*
Generate oh-my-opencode.jsonc config
*/}}
{{- define "ok8s.omoConfig" -}}
{
  "$schema": "https://raw.githubusercontent.com/code-yeongyu/oh-my-openagent/refs/heads/dev/assets/oh-my-opencode.schema.json",
  {{- if .Values.omo.enabled }}
  "sisyphus": {
    "enabled": true,
    "max_concurrent_tasks": {{ .Values.omo.sisyphus.maxConcurrentTasks }},
    "task_timeout": {{ .Values.omo.sisyphus.taskTimeout }}
  },
  "agents": {
    "oracle": {
      "enabled": true,
      "model": {{ .Values.omo.agents.oracle.model | quote }},
      "prompt_append": {{ .Values.omo.agents.oracle.promptAppend | quote }}
    },
    "librarian": {
      "enabled": true,
      "model": {{ .Values.omo.agents.librarian.model | quote }}
    }
  },
  "categories": {
    "quick": { "model": {{ .Values.omo.categories.quick.model | quote }} },
    "visual-engineering": { "model": {{ .Values.omo.categories.visualEngineering.model | quote }} }
  }
  {{- else }}
  "sisyphus": { "enabled": false }
  {{- end }}
}
{{- end }}

{{- define "ok8s.userList" -}}
{{- $names := list -}}
{{- range .Values.users }}
{{- $names = append $names .name -}}
{{- end }}
{{- join "," $names -}}
{{- end }}

{{- define "ok8s.userConfigChecksum" -}}
{{- $root := .root -}}
{{- $user := .user -}}
{{- toJson (dict "user" $user "sharedMcp" $root.Values.sharedMcp "sharedSkills" $root.Values.sharedSkills "omo" $root.Values.omo) -}}
{{- end }}

{{- define "ok8s.opencodeUserConfig" -}}
{{- $root := .root -}}
{{- $user := .user -}}
{{- $userMcp := $user.mcp | default (dict) -}}
{{- $userRemote := (get $userMcp "remote") | default (list) -}}
{{- $userLaptop := (get $userMcp "laptopServers") | default (list) -}}
{
  "$schema": "https://opencode.ai/config.json",
  {{- if $root.Values.plugins.enabled }}
  "plugin": [
    "oh-my-opencode@latest",
    "@tarquinen/opencode-dcp@latest"{{- range $root.Values.plugins.npm }},
    "{{ . }}"{{- end }}
  ],
  {{- end }}
  "mcp": {
    {{- $first := true }}
    {{- range $root.Values.sharedMcp }}
    {{- if not $first }},{{ end }}
    "{{ .name }}": {
      "type": "remote",
      "url": "{{ .url }}"{{ if .enabled }},
      "enabled": true{{ end }}{{ if .headers }},
      "headers": {{ .headers | toJson }}{{ end }}{{ if .oauth }},
      "oauth": {{ .oauth | toJson }}{{ end }}
    }
    {{- $first = false }}
    {{- end }}
    {{- range $userRemote }}
    {{- if not $first }},{{ end }}
    "{{ .name }}": {
      "type": "remote",
      "url": "{{ .url }}"{{ if .enabled }},
      "enabled": true{{ end }}{{ if .headers }},
      "headers": {{ .headers | toJson }}{{ end }}{{ if .oauth }},
      "oauth": {{ .oauth | toJson }}{{ end }}
    }
    {{- $first = false }}
    {{- end }}
    {{- range $userLaptop }}
    {{- if not $first }},{{ end }}
    "laptop_{{ $user.name }}_{{ .name }}": {
      "type": "remote",
      "url": "http://{{ $root.Release.Name }}-egress-{{ $user.name }}-{{ .name }}:{{ .port }}"{{ if .enabled }},
      "enabled": true{{ end }}
    }
    {{- $first = false }}
    {{- end }}
  }{{- $sharedSkills := $root.Values.sharedSkills | default (dict) }}{{- $userSkills := $user.skills | default (dict) }}{{- $npmSkills := concat (get $sharedSkills "npm" | default (list)) (get $userSkills "npm" | default (list)) }}{{- $configSkills := concat (get $sharedSkills "config" | default (list)) (get $userSkills "config" | default (list)) }}{{- if or ($npmSkills | len) ($configSkills | len) }},
  "skills": {
    "npm": {{ $npmSkills | toJson }},
    "config": {{ $configSkills | toJson }}
  }
  {{- end }}
}
{{- end }}

{{- define "ok8s.omoUserConfig" -}}
{{- $root := .root -}}
{{- $user := .user -}}
{{- $userSkills := $user.skills | default (dict) -}}
{{- $sharedSkills := $root.Values.sharedSkills | default (dict) -}}
{{- $sharedNpm := (get $sharedSkills "npm") | default (list) -}}
{{- $sharedConfig := (get $sharedSkills "config") | default (list) -}}
{{- $userNpm := (get $userSkills "npm") | default (list) -}}
{{- $userConfig := (get $userSkills "config") | default (list) -}}
{{- $npmSkills := concat $sharedNpm $userNpm -}}
{{- $configSkills := concat $sharedConfig $userConfig -}}
{
  "$schema": "https://raw.githubusercontent.com/code-yeongyu/oh-my-openagent/refs/heads/dev/assets/oh-my-opencode.schema.json",
  {{- if $root.Values.omo.enabled }}
  "sisyphus": {
    "enabled": true,
    "max_concurrent_tasks": {{ $root.Values.omo.sisyphus.maxConcurrentTasks }},
    "task_timeout": {{ $root.Values.omo.sisyphus.taskTimeout }}
  },
  "agents": {
    "oracle": {
      "enabled": true,
      "model": {{ $root.Values.omo.agents.oracle.model | quote }},
      "prompt_append": {{ $root.Values.omo.agents.oracle.promptAppend | quote }}
    },
    "librarian": {
      "enabled": true,
      "model": {{ $root.Values.omo.agents.librarian.model | quote }}
    }
  },
  "categories": {
    "quick": { "model": {{ $root.Values.omo.categories.quick.model | quote }} },
    "visual-engineering": { "model": {{ $root.Values.omo.categories.visualEngineering.model | quote }} }
  },
  "skills": {
    "npm": {{ $npmSkills | default (list) | toJson }},
    "config": {{ $configSkills | default (list) | toJson }}
  }
  {{- else }}
  "sisyphus": { "enabled": false },
  "skills": {
    "npm": {{ $npmSkills | default (list) | toJson }},
    "config": {{ $configSkills | default (list) | toJson }}
  }
  {{- end }}
}
{{- end }}
