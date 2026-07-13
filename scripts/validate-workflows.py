#!/usr/bin/env python3
"""
GitHub Actions Workflow Validator
Validates all workflow files in .github/workflows/ for:
  - Valid YAML syntax
  - Required top-level keys (name, on, jobs)
  - Jobs are properly nested under 'jobs:'
  - Steps have valid structure
  - Common formatting mistakes (missing spaces after colons)
  - Permissions key correctness
  - Uses references basic validity

Usage:
  python3 scripts/validate-workflows.py
  python3 scripts/validate-workflows.py --verbose
"""

import sys
import os
import glob
import re

try:
    import yaml
except ImportError:
    print("ERROR: PyYAML is required. Install with: pip install pyyaml")
    sys.exit(1)


# PyYAML (YAML 1.1) treats 'on', 'off', 'yes', 'no' as booleans.
# GitHub Actions uses 'on' as a trigger key, so we need a custom loader
# that treats only 'true'/'false' as booleans.
class GitHubWorkflowLoader(yaml.SafeLoader):
    pass

# Remove the default boolean resolver that matches 'on','off','yes','no','y','n','true','false'
# The fallback 'True in data' check in validate_workflow() is the primary defense
for ch in "YyNnOoTtFf":
    if ch in GitHubWorkflowLoader.yaml_implicit_resolvers:
        # Replace with our own that only matches true/false
        GitHubWorkflowLoader.yaml_implicit_resolvers[ch] = [
            ("tag:yaml.org,2002:bool", re.compile(r"^(?:true|false|TRUE|FALSE|True|False)$"))
        ]


def load_gh_workflow(stream):
    """Load a GitHub Actions workflow YAML file, handling the 'on' boolean issue."""
    return yaml.load(stream, Loader=GitHubWorkflowLoader)


VERBOSE = False

WORKFLOW_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), ".github", "workflows")

# Actions that exist but have known typos
KNOWN_TYPOS = {
    "drony/paths-filter": "dorny/paths-filter",
}


def log(msg, level="INFO"):
    if level == "ERROR":
        print(f"  ❌ {msg}")
    elif level == "WARN":
        print(f"  ⚠️  {msg}")
    elif level == "OK":
        print(f"  ✅ {msg}")
    elif level == "INFO" and VERBOSE:
        print(f"  ℹ️  {msg}")


def get_workflow_files():
    pattern = os.path.join(WORKFLOW_DIR, "*.yml")
    files = glob.glob(pattern)
    if not files:
        pattern = os.path.join(WORKFLOW_DIR, "*.yaml")
        files = glob.glob(pattern)
    return sorted(files)


def check_raw_yaml_issues(content, path):
    """Check for common YAML formatting mistakes that are valid YAML but wrong in GH Actions context."""
    issues = []
    lines = content.split("\n")

    for i, line in enumerate(lines, 1):
        stripped = line.lstrip()

        # Check for missing space after colon in key:value pairs
        # This catches patterns like `key:value` (no space after colon)
        colon_match = re.match(r"^(\s*[\w-]+):(\S)", stripped)
        if colon_match and not stripped.startswith("#"):
            key = colon_match.group(1).strip()
            value_start = colon_match.group(2)
            # Exclude known false positives like URLs, paths with ://, time formats
            if not any(ignore in value_start for ignore in ["//", "\\", "http", ":"]) and not re.match(r"^\d", value_start):
                issues.append((i, f"Missing space after colon: '{key}:{value_start}...' → should be '{key}: {value_start}...'"))

        # Check for missing space after dash in list items
        if re.match(r"^(\s*)-\w", stripped):
            issues.append((i, f"Missing space after dash: '{stripped[:40]}...' → should be '- ...'"))

        # Check for common typos in action names
        for typo, correct in KNOWN_TYPOS.items():
            if typo in line and "uses:" in stripped:
                issues.append((i, f"Action typo: '{typo}' → should be '{correct}'"))

    return issues


def validate_workflow(data, path):
    """Validate the parsed workflow structure."""
    errors = []
    warnings = []

    # 1. Check required top-level keys
    if "name" not in data:
        errors.append("Missing required top-level key: 'name'")

    # Handle 'on' key — PyYAML may parse it as bool True even with custom loader
    has_on = "on" in data
    has_on_as_bool = True in data if isinstance(data, dict) else False  # 'on' parsed as True
    if not has_on and not has_on_as_bool:
        errors.append("Missing required top-level key: 'on' (trigger events)")
    elif has_on:
        on_val = data["on"]
        if not isinstance(on_val, (str, dict, list)):
            errors.append("'on' should be a string, dict, or list of events")
    elif has_on_as_bool:
        # 'on' was parsed as boolean True — remap it for downstream validation
        on_val = data[True]
        data["on"] = on_val
        if not isinstance(on_val, (str, dict, list)):
            errors.append("'on' should be a string, dict, or list of events")
        warnings.append("'on' was parsed as a boolean by YAML loader — ensure your yaml library supports YAML 1.2 (consider using 'ruamel.yaml')")

    if "jobs" not in data:
        errors.append("Missing required top-level key: 'jobs'")
    elif not isinstance(data["jobs"], dict):
        errors.append("'jobs' must be a dictionary/mapping")
    elif len(data["jobs"]) == 0:
        warnings.append("'jobs' is empty — no jobs defined")

    # 2. Check permissions
    permissions = data.get("permissions", {})
    if isinstance(permissions, dict):
        valid_perms = {"actions", "checks", "contents", "deployments", "id-token",
                       "issues", "discussions", "packages", "pages", "pull-requests",
                       "repository-projects", "security-events", "statuses"}
        for perm_key in permissions:
            if perm_key not in valid_perms:
                warnings.append(f"Unknown permission key: '{perm_key}'")
            perm_val = permissions[perm_key]
            if isinstance(perm_val, str) and perm_val not in ("read", "write", "none"):
                warnings.append(f"Permission '{perm_key}' has unusual value: '{perm_val}' (expected: read, write, or none)")

    # 3. Validate each job
    jobs = data.get("jobs", {})
    if isinstance(jobs, dict):
        for job_name, job in jobs.items():
            validate_job(job_name, job, errors, warnings)

    return errors, warnings


def validate_job(job_name, job, errors, warnings):
    """Validate a single job definition."""
    # Fix: if a job value is a string (likely YAML parse quirk of a different structure being parsed wrong),
    # we can't validate it deeply
    if not isinstance(job, dict):
        errors.append(f"Job '{job_name}' is not a dictionary (YAML structure issue)")
        return

    # Check runs-on
    if "runs-on" not in job:
        errors.append(f"Job '{job_name}': missing 'runs-on'")

    # Check steps
    if "steps" not in job:
        errors.append(f"Job '{job_name}': missing 'steps'")
    elif not isinstance(job["steps"], list):
        errors.append(f"Job '{job_name}': 'steps' should be a list")
    else:
        for i, step in enumerate(job["steps"], 1):
            validate_step(job_name, i, step, errors, warnings)

    # Check needs
    if "needs" in job:
        needs = job["needs"]
        if isinstance(needs, str):
            pass  # single dependency - fine
        elif isinstance(needs, list):
            pass  # multiple dependencies - fine
        else:
            errors.append(f"Job '{job_name}': 'needs' should be a string or list")

    # Check if
    if "if" in job:
        if_cond = job["if"]
        if isinstance(if_cond, str):
            # Check for common escaped quote issues in conditions
            # e.g., if someone writes '[\"web\"]' in YAML single quotes,
            # the backslash is literal (\\" → one backslash + quote in the parsed string)
            if '\\"' in if_cond:
                warnings.append(f"Job '{job_name}': 'if' condition contains literal backslash-quote")
                warnings.append(f"  (likely an escaping error from YAML single quotes)")

    # Check strategy.matrix
    strategy = job.get("strategy", {})
    if isinstance(strategy, dict):
        matrix = strategy.get("matrix", {})
        if isinstance(matrix, dict) and "fail-fast" in strategy:
            ff = strategy["fail-fast"]
            if not isinstance(ff, bool):
                warnings.append(f"Job '{job_name}': 'strategy.fail-fast' should be a boolean")


def validate_step(job_name, step_num, step, errors, warnings):
    """Validate a single step."""
    if not isinstance(step, dict):
        errors.append(f"Job '{job_name}', step {step_num}: step is not a dictionary (YAML structure issue)")
        return

    has_uses = "uses" in step
    has_run = "run" in step

    if not has_uses and not has_run:
        errors.append(f"Job '{job_name}', step {step_num}: missing both 'uses' and 'run'")

    if has_uses:
        uses = step["uses"]
        if not isinstance(uses, str):
            errors.append(f"Job '{job_name}', step {step_num}: 'uses' should be a string")
        else:
            # Check action reference format: owner/repo@version
            if "@" not in uses:
                warnings.append(f"Job '{job_name}', step {step_num}: 'uses' value '{uses}' has no version (@tag) — not pinned")
            else:
                version_part = uses.split("@")[1]
                if version_part and not re.match(r"^[\w.\-/]+$", version_part):
                    warnings.append(f"Job '{job_name}', step {step_num}: unusual version tag '{version_part}'")

    if has_run:
        run_val = step["run"]
        if not isinstance(run_val, str):
            errors.append(f"Job '{job_name}', step {step_num}: 'run' should be a string")

    # Check name
    if "name" in step and not isinstance(step["name"], str):
        errors.append(f"Job '{job_name}', step {step_num}: 'name' should be a string")

    # Check if step has both uses and run (usually wrong)
    if has_uses and has_run:
        warnings.append(f"Job '{job_name}', step {step_num}: has both 'uses' and 'run' — only one should be present")


def validate_workflow_file(filepath):
    """Validate a single workflow file and return (filename, issues_count)."""
    filename = os.path.basename(filepath)
    print(f"\n📄 {filename}")

    try:
        with open(filepath, "r") as f:
            content = f.read()
    except IOError as e:
        log(f"Cannot read file: {e}", "ERROR")
        return filename, 1

    if not content.strip():
        log("File is empty", "WARN")
        return filename, 0

    # 1. Check raw YAML for formatting issues
    raw_issues = check_raw_yaml_issues(content, filepath)
    for line_num, msg in raw_issues:
        log(f"Line {line_num}: {msg}", "WARN")

    # 2. Parse YAML using GitHub-workflow-aware loader
    try:
        data = load_gh_workflow(content)
    except yaml.YAMLError as e:
        log(f"Invalid YAML: {e}", "ERROR")
        return filename, 1

    if data is None:
        log("File parsed as null (possibly only comments or empty)", "WARN")
        return filename, 0

    if not isinstance(data, dict):
        log(f"Root value is {type(data).__name__}, expected dict", "ERROR")
        return filename, 1

    # 3. Validate workflow structure
    errors, warnings = validate_workflow(data, filepath)

    for msg in errors:
        log(msg, "ERROR")
    for msg in warnings:
        log(msg, "WARN")

    total_issues = len(raw_issues) + len(errors) + len(warnings)

    if total_issues == 0:
        log("No issues found", "OK")

    return filename, total_issues


def main():
    global VERBOSE
    VERBOSE = "--verbose" in sys.argv or "-v" in sys.argv

    files = get_workflow_files()

    if not files:
        print(f"❌ No workflow files found in {WORKFLOW_DIR}")
        sys.exit(1)

    print(f"🔍 Found {len(files)} workflow file(s) to validate\n" + "=" * 50)

    total_issues = 0
    for filepath in files:
        _, issues = validate_workflow_file(filepath)
        total_issues += issues

    print("\n" + "=" * 50)
    if total_issues == 0:
        print(f"✅ All {len(files)} workflow file(s) passed validation — no issues found!")
        sys.exit(0)
    else:
        print(f"❌ Found {total_issues} total issue(s) across {len(files)} file(s)")
        sys.exit(1)


if __name__ == "__main__":
    main()
