import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { after, test } from 'node:test';
import { pathToFileURL } from 'node:url';
import ts from 'typescript';

// Use the existing compiler so the tests also run on the Node 20 build image.
const output = mkdtempSync(join(tmpdir(), 'rentalcore-dashboard-tests-'));
after(() => rmSync(output, { recursive: true, force: true }));
writeFileSync(join(output, 'package.json'), '{"type":"module"}');
for (const name of ['job-status', 'dashboard-schedule', 'dashboard-refresh']) {
  const source = readFileSync(new URL(`../src/lib/${name}.ts`, import.meta.url), 'utf8');
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.ESNext,
      rewriteRelativeImportExtensions: true,
    },
  });
  writeFileSync(join(output, `${name}.js`), compiled.outputText);
}
const { selectSchedule, startOfDay } = await import(pathToFileURL(join(output, 'dashboard-schedule.js')).href);
const { subscribeDashboardRefresh } = await import(pathToFileURL(join(output, 'dashboard-refresh.js')).href);

const today = new Date(2026, 9, 8);
const job = (jobID, startDate, endDate, status_id = 2) => ({
  jobID, startDate, endDate, status_id,
});
const ids = (jobs) => jobs.map(({ jobID }) => jobID);

test('overdue jobs cannot displace current and upcoming appointments', () => {
  const overdue = Array.from({ length: 8 }, (_, index) => job(index + 1, '2026-09-01', '2026-10-07'));
  const jobs = [...overdue, job(20, '2026-10-08', '2026-10-08'), job(21, '2026-10-09', '2026-10-10')];
  assert.deepEqual(ids(selectSchedule(jobs, today)), [20, 21]);
});

test('completed, cancelled and undated jobs stay out of the radar', () => {
  const jobs = [
    job(1, '2026-10-08', '2026-10-09', 4),
    job(2, '2026-10-08', '2026-10-09', 6),
    job(3, null, null),
    job(4, 'invalid', 'invalid'),
    job(5, '2026-10-09', '2026-10-10', 1),
  ];
  assert.deepEqual(ids(selectSchedule(jobs, today)), [5]);
});

test('running jobs precede the next five appointments without changing the source list', () => {
  const jobs = [
    job(8, '2026-10-15'), job(7, '2026-10-14'), job(6, '2026-10-13'),
    job(5, '2026-10-12'), job(4, '2026-10-11'), job(3, '2026-10-10'),
    job(2, '2026-10-09'), job(1, '2026-10-07', '2026-10-09'),
  ];
  assert.deepEqual(ids(selectSchedule(jobs, today)), [1, 2, 3, 4, 5]);
  assert.deepEqual(ids(jobs), [8, 7, 6, 5, 4, 3, 2, 1]);
});

test('a job ending today leaves the radar after the local day changes', () => {
  const jobs = [job(1, '2026-10-07', '2026-10-08'), job(2, '2026-10-09', '2026-10-10')];
  assert.deepEqual(ids(selectSchedule(jobs, today)), [1, 2]);
  assert.deepEqual(ids(selectSchedule(jobs, new Date(2026, 9, 9))), [2]);
});

test('end-only dates are ordered with upcoming jobs', () => {
  const jobs = [job(3, '2026-10-12'), job(2, null, '2026-10-10'), job(1, null, '2026-10-07')];
  assert.deepEqual(ids(selectSchedule(jobs, today)), [2, 3]);
});

test('day boundaries use the local calendar date', () => {
  const date = new Date(2026, 9, 8, 23, 59);
  assert.equal(startOfDay(date).getTime(), today.getTime());
});

function browser(t) {
  t.mock.timers.enable({ apis: ['setInterval'] });
  const window = new EventTarget();
  window.setInterval = (...args) => setInterval(...args);
  window.clearInterval = (id) => clearInterval(id);
  const document = new EventTarget();
  document.visibilityState = 'visible';
  globalThis.window = window;
  globalThis.document = document;
  t.after(() => {
    delete globalThis.window;
    delete globalThis.document;
  });
  return { window, document };
}

test('visible dashboards refresh every minute and on focus', (t) => {
  const { window } = browser(t);
  let calls = 0;
  const unsubscribe = subscribeDashboardRefresh(() => calls++);
  t.mock.timers.tick(59_999);
  assert.equal(calls, 0);
  t.mock.timers.tick(1);
  assert.equal(calls, 1);
  window.dispatchEvent(new Event('focus'));
  assert.equal(calls, 2);
  unsubscribe();
});

test('hidden tabs pause refresh and reload when visible again', (t) => {
  const { window, document } = browser(t);
  let calls = 0;
  const unsubscribe = subscribeDashboardRefresh(() => calls++);
  document.visibilityState = 'hidden';
  t.mock.timers.tick(120_000);
  window.dispatchEvent(new Event('focus'));
  document.dispatchEvent(new Event('visibilitychange'));
  assert.equal(calls, 0);
  document.visibilityState = 'visible';
  document.dispatchEvent(new Event('visibilitychange'));
  assert.equal(calls, 1);
  unsubscribe();
});

test('leaving the dashboard removes timers and browser listeners', (t) => {
  const { window, document } = browser(t);
  let calls = 0;
  const unsubscribe = subscribeDashboardRefresh(() => calls++);
  unsubscribe();
  t.mock.timers.tick(120_000);
  window.dispatchEvent(new Event('focus'));
  document.dispatchEvent(new Event('visibilitychange'));
  assert.equal(calls, 0);
});
