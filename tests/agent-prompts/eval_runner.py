#!/usr/bin/env python3
"""
EPO CLI Agent UAT Runner

Runs each EPO prompt through frix-headless and captures structured evaluation
results written by the agent to per-prompt JSON files.

Usage:
    python tests/agent-prompts/eval_runner.py
    python tests/agent-prompts/eval_runner.py --prompts 1,3,10
    python tests/agent-prompts/eval_runner.py --dry-run

Output:
    tests/agent-prompts/results/prompt01.json ... prompt10.json
    tests/agent-prompts/results/eval_summary.json
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Optional


SCRIPT_DIR = Path(__file__).parent.resolve()
REPO_ROOT = SCRIPT_DIR.parent.parent.resolve()
DEFAULT_FRIX_ROOT_ENV = "FRIX_ROOT"
RESULTS_DIR = SCRIPT_DIR / "results"
WORKSPACE_BASE = SCRIPT_DIR / "workspaces"
P10_MIN_TIMEOUT = 900


OUTPUT_INSTRUCTION_TEMPLATE = """\
FINAL STEP — REQUIRED: When you complete the task, write your evaluation
result as valid JSON to this file (mandatory):

File path: {eval_file}

Use this exact schema:
{{
  "success": true or false,
  "turns": <integer count of CLI command invocations>,
  "tools_used_count": <integer count of tool calls>,
  "commands": ["epo ...", "epo ..."],
  "features_tested": ["pub search", "family get", "..."],
  "ax": "your experience using the CLI; what worked vs friction",
  "suggestions": "specific CLI/skill improvements for agent usability",
  "blocking_issues": ["issue 1", "issue 2"]
}}
"""


CONSTRAINT = (
    "IMPORTANT: You may NOT use any MineSoft tools, MineSoft patent APIs, "
    "or any non-EPO patent data tools. Use only the `epo` CLI for patent data. "
    "This is agent UAT of the EPO CLI itself, not user-task completion."
)


@dataclass
class PromptDef:
    number: int
    file_name: str
    prompt_text: str
    full_prompt: str
    eval_file: Path


def find_frix_headless(frix_root: Path) -> Optional[Path]:
    candidates = [
        frix_root / "frix_headless.py",
        frix_root / "frix-headless.py",
    ]
    for path in candidates:
        if path.exists():
            return path
    return None


def resolve_frix_root(flag_value: str) -> Path:
    if flag_value.strip():
        return Path(flag_value).expanduser().resolve()

    env_value = os.getenv(DEFAULT_FRIX_ROOT_ENV, "").strip()
    if env_value:
        return Path(env_value).expanduser().resolve()

    candidate = (REPO_ROOT.parent / "frix-agent").resolve()
    if candidate.exists():
        return candidate

    raise RuntimeError(
        "FRIX root not configured. Pass --frix-root or set FRIX_ROOT "
        "(for example: $env:FRIX_ROOT='C:\\path\\to\\frix-agent')."
    )


def extract_prompt_text(md_path: Path) -> str:
    text = md_path.read_text(encoding="utf-8")
    in_prompt = False
    lines: list[str] = []

    for line in text.splitlines():
        if line.strip().startswith("## The Prompt"):
            in_prompt = True
            continue
        if in_prompt:
            if line.strip().startswith("## "):
                break
            stripped = line.strip()
            if stripped.startswith("> "):
                lines.append(stripped[2:])
            elif stripped == ">":
                lines.append("")
            elif stripped == "" and lines:
                lines.append("")
            elif lines and not stripped.startswith(">"):
                if stripped:
                    break

    return "\n".join(lines).strip()


def build_full_prompt(prompt_text: str, prompt_num: int, eval_file: Path) -> str:
    output_instruction = OUTPUT_INSTRUCTION_TEMPLATE.format(eval_file=str(eval_file))
    return (
        f"/EPO\n\n"
        f"{CONSTRAINT}\n\n"
        f"--- BEGIN TASK (Prompt {prompt_num:02d}) ---\n\n"
        f"{prompt_text}\n\n"
        f"--- END TASK ---\n\n"
        f"{output_instruction}"
    )


def run_command(args: list[str], cwd: Optional[Path] = None) -> tuple[int, str, str]:
    proc = subprocess.run(
        args,
        cwd=str(cwd) if cwd else None,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    return proc.returncode, proc.stdout, proc.stderr


def preflight_check(epo_bin: str) -> None:
    skill_path = REPO_ROOT / "skills" / "epo" / "SKILL.md"
    if not skill_path.exists():
        raise RuntimeError(f"Missing skill file: {skill_path}")

    code, out, err = run_command([epo_bin, "--version"])
    if code != 0:
        raise RuntimeError(
            "Could not run `epo --version`. Ensure epo binary is built and on PATH.\n"
            f"stdout: {out.strip()}\n"
            f"stderr: {err.strip()}"
        )


def read_eval_file(eval_file: Path) -> Optional[dict]:
    if not eval_file.exists():
        return None
    try:
        text = eval_file.read_text(encoding="utf-8").strip()
        if text.startswith("```"):
            lines = text.splitlines()
            if lines and lines[0].startswith("```"):
                lines = lines[1:]
            if lines and lines[-1].startswith("```"):
                lines = lines[:-1]
            text = "\n".join(lines).strip()
        return json.loads(text)
    except Exception:
        return None


def parse_eval_from_stdout(stdout: str) -> Optional[dict]:
    for line in stdout.splitlines():
        line = line.strip()
        if line.startswith("EVAL_RESULT_JSON:"):
            json_str = line[len("EVAL_RESULT_JSON:") :].strip()
            try:
                return json.loads(json_str)
            except json.JSONDecodeError:
                pass

    pattern = re.compile(
        r'\{[^{}]*"success"\s*:\s*(true|false)[^{}]*"commands"\s*:\s*\[.*?\][^{}]*\}',
        re.DOTALL,
    )
    match = pattern.search(stdout)
    if not match:
        return None
    try:
        return json.loads(match.group())
    except json.JSONDecodeError:
        return None


def save_log(prompt_num: int, full_prompt: str, result: dict) -> None:
    log_dir = RESULTS_DIR / "logs"
    log_dir.mkdir(parents=True, exist_ok=True)
    path = log_dir / f"prompt{prompt_num:02d}.log"
    with open(path, "w", encoding="utf-8") as f:
        f.write(f"Prompt {prompt_num:02d}\n")
        f.write(f"Timestamp: {result['timestamp']}\n")
        f.write(f"Duration: {result['duration_seconds']}s\n")
        f.write(f"Exit Code: {result['exit_code']}\n")
        f.write(f"\n{'='*70}\nFULL PROMPT\n{'='*70}\n")
        f.write(full_prompt)
        f.write(f"\n\n{'='*70}\nSTDOUT (tail)\n{'='*70}\n")
        f.write(result.get("raw_stdout_tail", "(empty)"))
        f.write(f"\n\n{'='*70}\nSTDERR (tail)\n{'='*70}\n")
        f.write(result.get("raw_stderr_tail", "(empty)"))


def run_prompt(
    p: PromptDef,
    frix_root: Path,
    frix_headless: Path,
    timeout: int,
    verbose: bool,
) -> dict:
    workspace = WORKSPACE_BASE / f"prompt{p.number:02d}"
    workspace.mkdir(parents=True, exist_ok=True)
    if p.eval_file.exists():
        p.eval_file.unlink()

    result = {
        "prompt_number": p.number,
        "prompt_file": p.file_name,
        "prompt_text": p.prompt_text,
        "timestamp": datetime.now().isoformat(),
        "duration_seconds": 0,
        "exit_code": None,
        "eval_result": None,
        "raw_stdout_tail": "",
        "raw_stderr_tail": "",
        "parse_error": None,
    }

    print(f"  [{p.number:02d}/10] Running...", end=" ", flush=True)
    start = time.time()
    try:
        env = os.environ.copy()
        env["PYTHONIOENCODING"] = "utf-8"
        proc = subprocess.run(
            [sys.executable, str(frix_headless), p.full_prompt, "--workspace", str(workspace), "--verbose"],
            cwd=str(frix_root),
            env=env,
            capture_output=True,
            text=True,
            timeout=timeout,
            encoding="utf-8",
            errors="replace",
        )
        duration = time.time() - start
        result["duration_seconds"] = round(duration, 2)
        result["exit_code"] = proc.returncode
        result["raw_stdout_tail"] = proc.stdout[-4000:] if proc.stdout else ""
        result["raw_stderr_tail"] = proc.stderr[-1500:] if proc.stderr else ""

        eval_result = read_eval_file(p.eval_file)
        if not eval_result:
            eval_result = parse_eval_from_stdout(proc.stdout)

        if eval_result:
            result["eval_result"] = eval_result
            status = "PASS" if eval_result.get("success") else "FAIL"
        else:
            result["parse_error"] = (
                f"Agent did not write eval JSON to {p.eval_file.name} "
                "and no structured fallback found in stdout."
            )
            status = "NO_RESULT"
        print(f"{status:10} ({duration:.0f}s)")

        if verbose and eval_result:
            print(
                f"           turns={eval_result.get('turns', '?')}, "
                f"tools={eval_result.get('tools_used_count', '?')}, "
                f"commands={len(eval_result.get('commands', []))}"
            )
    except subprocess.TimeoutExpired:
        duration = time.time() - start
        result["duration_seconds"] = round(duration, 2)
        result["exit_code"] = -1
        result["parse_error"] = f"Timed out after {timeout}s"
        eval_result = read_eval_file(p.eval_file)
        if eval_result:
            result["eval_result"] = eval_result
        print(f"TIMEOUT    ({timeout}s)")
    except Exception as exc:
        duration = time.time() - start
        result["duration_seconds"] = round(duration, 2)
        result["exit_code"] = -2
        result["parse_error"] = str(exc)
        print(f"ERROR      ({duration:.0f}s) {exc}")

    return result


def to_int_set(csv_value: str) -> set[int]:
    out: set[int] = set()
    for raw in csv_value.split(","):
        raw = raw.strip()
        if not raw:
            continue
        out.add(int(raw))
    return out


def main() -> None:
    parser = argparse.ArgumentParser(description="EPO CLI Agent UAT Runner")
    parser.add_argument("--prompts", type=str, default=None, help="Comma-separated prompt numbers (e.g. 1,3,10)")
    parser.add_argument("--timeout", type=int, default=700, help="Base timeout per prompt in seconds (default: 700)")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose per-prompt details")
    parser.add_argument("--dry-run", action="store_true", help="Print full prompts without running")
    parser.add_argument(
        "--frix-root",
        type=str,
        default="",
        help="Path to frix-agent repo (or set FRIX_ROOT)",
    )
    parser.add_argument("--epo-bin", type=str, default="epo", help="EPO binary name/path (default: epo)")
    parser.add_argument("--skip-preflight", action="store_true", help="Skip preflight checks (skill and binary)")
    args = parser.parse_args()

    try:
        frix_root = resolve_frix_root(args.frix_root)
    except Exception as exc:
        print(f"Error: {exc}", file=sys.stderr)
        sys.exit(1)
    frix_headless = find_frix_headless(frix_root)
    if not frix_headless:
        print(f"Error: frix headless script not found in {frix_root}", file=sys.stderr)
        sys.exit(1)

    if not args.skip_preflight:
        try:
            preflight_check(args.epo_bin)
        except Exception as exc:
            print(f"Preflight failed: {exc}", file=sys.stderr)
            sys.exit(1)

    prompt_files = sorted(SCRIPT_DIR.glob("[0-9][0-9]-*.md"))
    if not prompt_files:
        print(f"Error: no prompt markdown files found in {SCRIPT_DIR}", file=sys.stderr)
        sys.exit(1)

    if args.prompts:
        wanted = to_int_set(args.prompts)
        prompt_files = [p for p in prompt_files if int(p.name[:2]) in wanted]
        if not prompt_files:
            print("Error: no prompts matched --prompts selection", file=sys.stderr)
            sys.exit(1)

    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    prompts: list[PromptDef] = []
    for pf in prompt_files:
        num = int(pf.name[:2])
        text = extract_prompt_text(pf)
        eval_file = RESULTS_DIR / f"prompt{num:02d}_eval.json"
        full = build_full_prompt(text, num, eval_file.resolve())
        prompts.append(
            PromptDef(
                number=num,
                file_name=pf.name,
                prompt_text=text,
                full_prompt=full,
                eval_file=eval_file,
            )
        )

    if args.dry_run:
        for p in prompts:
            print(f"\n{'='*70}")
            print(f"PROMPT {p.number:02d} ({p.file_name})")
            print(f"{'='*70}\n")
            print(p.full_prompt)
        print(f"\n{len(prompts)} prompts prepared (dry-run only).")
        return

    print(f"\n{'='*70}")
    print("EPO CLI AGENT UAT")
    print(f"{'='*70}")
    print(f"Prompts:   {len(prompts)}")
    print(f"Timeout:   {args.timeout}s base per prompt (P10 min {P10_MIN_TIMEOUT}s)")
    print(f"Results:   {RESULTS_DIR}")
    print(f"Workspace: {WORKSPACE_BASE}")
    print(f"FRIX:      {frix_headless}")
    print(f"{'='*70}\n")

    all_results: list[dict] = []
    start_all = time.time()
    for p in prompts:
        timeout = args.timeout
        if p.number == 10 and timeout < P10_MIN_TIMEOUT:
            timeout = P10_MIN_TIMEOUT
        result = run_prompt(p, frix_root, frix_headless, timeout, args.verbose)
        all_results.append(result)

        clean = {k: v for k, v in result.items() if not k.startswith("raw_")}
        with open(RESULTS_DIR / f"prompt{p.number:02d}.json", "w", encoding="utf-8") as f:
            json.dump(clean, f, indent=2)
        save_log(p.number, p.full_prompt, result)

    total_duration = time.time() - start_all

    passed = [r for r in all_results if (r.get("eval_result") or {}).get("success")]
    failed = [r for r in all_results if r.get("eval_result") and not r["eval_result"].get("success")]
    no_result = [r for r in all_results if not r.get("eval_result")]

    all_commands: list[str] = []
    all_features: list[str] = []
    all_ax: list[str] = []
    all_suggestions: list[str] = []
    all_blockers: list[str] = []

    for r in all_results:
        er = r.get("eval_result") or {}
        all_commands.extend(er.get("commands", []))
        all_features.extend(er.get("features_tested", []))
        if er.get("ax"):
            all_ax.append(f"P{r['prompt_number']:02d}: {er['ax']}")
        if er.get("suggestions"):
            all_suggestions.append(f"P{r['prompt_number']:02d}: {er['suggestions']}")
        for blocker in er.get("blocking_issues", []):
            all_blockers.append(f"P{r['prompt_number']:02d}: {blocker}")

    summary = {
        "evaluation": "epo-skill",
        "timestamp": datetime.now().isoformat(),
        "total_prompts": len(all_results),
        "passed": len(passed),
        "failed": len(failed),
        "no_result": len(no_result),
        "pass_rate": f"{(len(passed) / len(all_results) * 100):.0f}%" if all_results else "0%",
        "total_duration_seconds": round(total_duration, 2),
        "total_command_invocations": len(all_commands),
        "unique_commands_used": sorted(set(all_commands)),
        "features_tested": sorted(set(all_features)),
        "agent_experience": all_ax,
        "agent_suggestions": all_suggestions,
        "blocking_issues": all_blockers,
        "per_prompt": [
            {
                "prompt": r["prompt_number"],
                "success": (r.get("eval_result") or {}).get("success"),
                "turns": (r.get("eval_result") or {}).get("turns"),
                "tools_used_count": (r.get("eval_result") or {}).get("tools_used_count"),
                "commands_count": len((r.get("eval_result") or {}).get("commands", [])),
                "duration_seconds": r["duration_seconds"],
            }
            for r in all_results
        ],
    }

    summary_file = RESULTS_DIR / "eval_summary.json"
    with open(summary_file, "w", encoding="utf-8") as f:
        json.dump(summary, f, indent=2)

    print(f"\n{'='*70}")
    print("EVALUATION SUMMARY")
    print(f"{'='*70}")
    print(f"Passed:     {len(passed)}/{len(all_results)}")
    print(f"Failed:     {len(failed)}/{len(all_results)}")
    print(f"No Result:  {len(no_result)}/{len(all_results)}")
    print(f"Duration:   {total_duration:.0f}s total")
    print(f"Commands:   {len(all_commands)} invocations, {len(set(all_commands))} unique")
    print(f"{'='*70}")
    print(f"\nResults:    {RESULTS_DIR}")
    print(f"Summary:    {summary_file}")

    if no_result:
        print("\nPrompts with no parseable result:")
        for r in no_result:
            print(f"  P{r['prompt_number']:02d}: {r.get('parse_error', 'unknown')}")

    if all_suggestions:
        print("\nAgent Suggestions (first 5):")
        for s in all_suggestions[:5]:
            print(f"  {s[:160]}")
    print()


if __name__ == "__main__":
    main()
