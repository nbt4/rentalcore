import axios from 'axios';

const API_BASE_URL = '/api/v1';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

// ── Types ────────────────────────────────────────────────

export interface Customer {
  customer_id: number;
  companyname?: string | null;
  lastname?: string | null;
  firstname?: string | null;
  street?: string | null;
  housenumber?: string | null;
  ZIP?: string | null;
  city?: string | null;
  phonenumber?: string | null;
  email?: string | null;
  customertype?: string | null;
  notes?: string | null;
}

export function customerDisplayName(c: Customer): string {
  if (c.companyname) return c.companyname;
  if (c.firstname && c.lastname) return `${c.firstname} ${c.lastname}`;
  return c.lastname || c.firstname || 'Unbekannt';
}

export interface JobStatus {
  status_id: number;
  status: string;
}

export interface JobDevice {
  jobID: number;
  deviceID: string;
  device: {
    deviceID: string;
    serialnumber?: string;
    product?: {
      name?: string;
      itemcostperday?: number;
    };
  };
  custom_price?: number | null;
}

export interface Job {
  jobID: number;
  job_code: string;
  customer_id: number;
  customer?: Customer;
  status_id: number;
  status?: JobStatus;
  description?: string | null;
  startDate?: string | null;
  endDate?: string | null;
  revenue?: number;
  final_revenue?: number | null;
  device_count?: number;
  created_at?: string;
}

export interface DashboardStats {
  total_jobs: number;
  active_jobs: number;
  total_customers: number;
  revenue_month: number;
  jobs_this_month: number;
}

// ── API functions ────────────────────────────────────────

export const jobsApi = {
  getAll: (params?: Record<string, string | number>) =>
    api.get<{ jobs: Job[] }>('/jobs', { params }),
  getById: (id: number) =>
    api.get<Job>(`/jobs/${id}`),
  getDevices: (id: number) =>
    api.get<{ devices: JobDevice[] }>(`/jobs/${id}/devices`),
  create: (data: Partial<Job>) =>
    api.post<{ jobID: number; job_code: string }>('/jobs', data),
  update: (id: number, data: Partial<Job>) =>
    api.put<{ message: string }>(`/jobs/${id}`, data),
  delete: (id: number) =>
    api.delete<{ message: string }>(`/jobs/${id}`),
};

export const customersApi = {
  getAll: (params?: Record<string, string>) =>
    api.get<{ customers: Customer[] }>('/customers', { params }),
  getById: (id: number) =>
    api.get<Customer>(`/customers/${id}`),
  create: (data: Partial<Customer>) =>
    api.post<{ customer_id: number }>('/customers', data),
  update: (id: number, data: Partial<Customer>) =>
    api.put<{ message: string }>(`/customers/${id}`, data),
  delete: (id: number) =>
    api.delete<{ message: string }>(`/customers/${id}`),
};

export const statusApi = {
  getAll: () => api.get<{ statuses: JobStatus[] }>('/statuses'),
};

export const analyticsApi = {
  getRevenue: (params?: Record<string, string>) =>
    api.get('/analytics/revenue', { params }),
  getEquipment: () =>
    api.get('/analytics/equipment'),
};

export const usersApi = {
  getAll: () => api.get('/security/auth/users'),
};
