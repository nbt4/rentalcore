export const JOB_STATUS_IDS = {
  planning: 1,
  confirmed: 2,
  completed: 4,
  cancelled: 6,
} as const;

export function isFinishedJob(statusId: number): boolean {
  return statusId === JOB_STATUS_IDS.completed || statusId === JOB_STATUS_IDS.cancelled;
}
