import type { Job } from './api';
import { isFinishedJob } from './job-status.ts';

export function startOfDay(date = new Date()) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

export function parseDate(value?: string | null) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function selectSchedule(jobs: Job[], today: Date) {
  return jobs
    .filter((job) => {
      if (isFinishedJob(job.status_id)) return false;
      const start = parseDate(job.startDate);
      const end = parseDate(job.endDate);
      return Boolean((start || end) && (!end || startOfDay(end) >= today));
    })
    .sort((a, b) => {
      const aDate = parseDate(a.startDate) || parseDate(a.endDate);
      const bDate = parseDate(b.startDate) || parseDate(b.endDate);
      return aDate!.getTime() - bDate!.getTime() || a.jobID - b.jobID;
    })
    .slice(0, 5);
}
