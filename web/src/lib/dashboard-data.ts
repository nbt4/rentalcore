import type { Customer, Job } from './api';

interface DashboardClient {
  get<T>(path: string, options: { signal: AbortSignal }): Promise<{ data: T }>;
}

export async function loadDashboardData(client: DashboardClient, controller: AbortController) {
  try {
    const [jobsRes, customersRes] = await Promise.all([
      client.get<{ jobs: Job[] }>('/jobs', { signal: controller.signal }),
      client.get<{ customers: Customer[] }>('/customers', { signal: controller.signal }),
    ]);
    return { jobs: jobsRes.data.jobs || [], customers: customersRes.data.customers || [] };
  } catch (error) {
    controller.abort();
    throw error;
  }
}
