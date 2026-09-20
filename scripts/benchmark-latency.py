"""Measure uncached end-to-end Reaper latency on generated Git patches."""
import argparse
import json
import os
from pathlib import Path
import statistics
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser()
parser.add_argument('--binary', required=True)
parser.add_argument('--output', default='benchmarks/latency.json')
args = parser.parse_args()
binary = str(Path(args.binary).resolve())
results = []
for size, count in [('small', 1), ('medium', 8), ('large', 32)]:
    with tempfile.TemporaryDirectory(prefix='reaper-latency-') as directory:
        root = Path(directory)
        def git(*argv):
            subprocess.run(['git', '-C', directory, *argv], check=True, capture_output=True)
        git('init')
        for i in range(count):
            (root / f'case{i}.go').write_text(f'package example\nfunc Value{i}() int {{ return 1 }}\n')
        git('add', '.')
        git('-c', 'user.name=Reaper benchmark', '-c', 'user.email=benchmark@example.invalid', 'commit', '-m', 'Fixture')
        for i in range(count):
            (root / f'case{i}.go').write_text(f'package example\nfunc Value{i}() int {{ return 2 }}\n')
        samples = []
        for _ in range(3):
            start = time.perf_counter()
            run = subprocess.run([binary, 'check', '--no-cache', '--provider', 'openrouter', '--model', 'typesafe/jev-1.13', '--task', 'Change each constant from 1 to 2', '--format', 'json'], cwd=directory, capture_output=True, text=True)
            elapsed = time.perf_counter() - start
            if run.returncode not in (0, 1):
                raise RuntimeError(f'{size} benchmark did not complete (exit {run.returncode})')
            report = json.loads(run.stdout)
            if not report['complete']:
                raise RuntimeError('Incomplete latency sample')
            samples.append(elapsed)
        results.append({'size': size, 'changed_files': count, 'samples_seconds': samples, 'median_seconds': statistics.median(samples)})
Path(args.output).write_text(json.dumps({'provider': 'openrouter', 'model': 'typesafe/jev-1.13', 'platform': os.name, 'cache': False, 'provenance': 'Generated one-line function changes; three wall-clock samples per size, not a production workload.', 'results': results}, indent=2) + '\n')
