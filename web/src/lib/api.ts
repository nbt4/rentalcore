import axios from 'axios';
import { appPath } from './app-paths';

const API_BASE_URL = appPath('/api/v1');

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
  is_customer?: boolean;
  is_supplier?: boolean;
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
  venue_id?: number | null;
  venue?: Venue | null;
}

export interface DashboardStats {
  total_jobs: number;
  active_jobs: number;
  total_customers: number;
  revenue_month: number;
  jobs_this_month: number;
}

export interface RevenueDrilldownNode {
  id: string;
  type: 'category' | 'product' | 'rental_product' | 'service' | 'package' | 'device';
  label: string;
  revenue: number;
  net_revenue: number;
  gross_revenue: number;
  tax_amount: number;
  cost: number;
  margin: number;
  margin_percent: number;
  has_cost: boolean;
  quantity: number;
  bookings: number;
  children: RevenueDrilldownNode[];
}

export interface RevenueDrilldown {
  period: string;
  start_date?: string;
  end_date?: string;
  total_revenue: number;
  total_net_revenue: number;
  total_gross_revenue: number;
  total_tax_amount: number;
  attributed_revenue: number;
  attributed_net_revenue: number;
  attributed_gross_revenue: number;
  unattributed_revenue: number;
  rental_revenue: number;
  rental_net_revenue: number;
  rental_gross_revenue: number;
  rental_cost: number;
  rental_margin: number;
  rental_margin_percent: number;
  job_count: number;
  categories: RevenueDrilldownNode[];
  monthly_revenue: Array<{
    month: string;
    net_revenue: number;
    gross_revenue: number;
    job_count: number;
  }>;
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
  getRevenueDrilldown: (period = 'all') =>
    api.get<RevenueDrilldown>('/analytics/revenue/drilldown', { params: { period } }),
  getEquipment: () =>
    api.get('/analytics/equipment'),
};

export const usersApi = {
  getAll: () => api.get('/security/auth/users'),
};

// ── Position Types ────────────────────────────────────────

export interface JobPositionDevice {
  id: number;
  position_id: number;
  device_id: string;
  scanned_at: string;
  scanned_by: string;
}

export interface JobPosition {
  position_id: number;
  job_id: number;
  position_type: 'product' | 'service' | 'rental' | 'package';
  product_id: number | null;
  service_item_id: number | null;
  rental_equipment_id: number | null;
  description: string;
  quantity: number;
  unit: string;
  unit_price: number;
  follow_day_factor: number;
  discount_percent: number;
  discount_amount: number;
  tax_rate: number;
  sort_order: number;
  product?: { productID: number; name: string; itemcostperday?: number } | null;
  service_item?: { id: number; name: string; default_price?: number; unit?: string } | null;
  rental_equipment?: { equipmentID: number; productName: string; rentalPrice?: number } | null;
  devices?: JobPositionDevice[];
}

export interface JobTotals {
  event_days: number;
  subtotal: number;
  global_discount: number;
  netto: number;
  tax_rate: number;
  tax: number;
  brutto: number;
  multiply_by_days: boolean;
  prices_include_tax: boolean;
}

export interface RentalCatalogItem {
  equipmentID: number;
  productName: string;
  supplierName: string;
  rentalPrice: number;
  category: string;
}

export interface Skill {
  id: number;
  name: string;
  category: string;
  description?: string;
}

export interface Employee {
  id: number;
  first_name: string;
  last_name: string;
  email?: string | null;
  phone?: string | null;
  mobile?: string | null;
  street?: string | null;
  house_number?: string | null;
  zip?: string | null;
  city?: string | null;
  country?: string | null;
  date_of_birth?: string | null;
  iban?: string | null;
  notes?: string | null;
  is_active: boolean;
  skills?: Skill[];
}

export interface JobEmployee {
  job_id: number;
  employee_id: number;
  role?: string;
  employee: Employee;
}

export const skillsApi = {
  list: () => api.get<Skill[]>('/skills'),
  create: (data: Partial<Skill>) => api.post<Skill>('/skills', data),
  update: (id: number, data: Partial<Skill>) => api.put<Skill>(`/skills/${id}`, data),
  delete: (id: number) => api.delete(`/skills/${id}`),
};

export const employeesApi = {
  list: () => api.get<Employee[]>('/employees'),
  listActive: () => api.get<Employee[]>('/employees/active'),
  getById: (id: number) => api.get<Employee>(`/employees/${id}`),
  create: (data: Partial<Employee> & { skill_ids?: number[] }) => api.post<Employee>('/employees', data),
  update: (id: number, data: Partial<Employee> & { skill_ids?: number[] }) => api.put<Employee>(`/employees/${id}`, data),
  delete: (id: number) => api.delete(`/employees/${id}`),
};

export const jobEmployeesApi = {
  list: (jobId: number) => api.get<JobEmployee[]>(`/jobs/${jobId}/employees`),
  assign: (jobId: number, employeeId: number, role?: string) =>
    api.post(`/jobs/${jobId}/employees`, { employee_id: employeeId, role }),
  remove: (jobId: number, employeeId: number) =>
    api.delete(`/jobs/${jobId}/employees/${employeeId}`),
};

export interface Venue {
  id: number;
  name: string;
  street?: string | null;
  house_number?: string | null;
  zip?: string | null;
  city?: string | null;
  contact_name?: string | null;
  phone?: string | null;
  email?: string | null;
  notes?: string | null;
}

export const venuesApi = {
  list: () => api.get<Venue[]>('/venues'),
  create: (data: Omit<Venue, 'id'>) => api.post<Venue>('/venues', data),
  update: (id: number, data: Omit<Venue, 'id'>) => api.put<Venue>(`/venues/${id}`, data),
  delete: (id: number) => api.delete(`/venues/${id}`),
};

export const positionsApi = {
  getByJob: (jobId: number) =>
    api.get<{ positions: JobPosition[] }>(`/jobs/${jobId}/positions`),
  create: (jobId: number, data: Partial<JobPosition>) =>
    api.post<{ position: JobPosition }>(`/jobs/${jobId}/positions`, data),
  update: (jobId: number, posId: number, data: Partial<JobPosition>) =>
    api.put<{ position: JobPosition }>(`/jobs/${jobId}/positions/${posId}`, data),
  delete: (jobId: number, posId: number) =>
    api.delete(`/jobs/${jobId}/positions/${posId}`),
  reorder: (jobId: number, positionIds: number[]) =>
    api.patch(`/jobs/${jobId}/positions/reorder`, { position_ids: positionIds }),
  assignDevice: (jobId: number, posId: number, deviceId: string, scannedBy?: string) =>
    api.post(`/jobs/${jobId}/positions/${posId}/devices`, { device_id: deviceId, scanned_by: scannedBy || '' }),
  removeDevice: (jobId: number, posId: number, deviceId: string) =>
    api.delete(`/jobs/${jobId}/positions/${posId}/devices/${deviceId}`),
  getTotals: (jobId: number) =>
    api.get<JobTotals>(`/jobs/${jobId}/totals`),
  updatePriceSettings: (jobId: number, settings: { multiply_by_days?: boolean; prices_include_tax?: boolean }) =>
    api.patch(`/jobs/${jobId}/price-settings`, settings),
  getRentalCatalog: () =>
    api.get<{ items: RentalCatalogItem[] }>('/rental-catalog'),
};
