import axios from "axios";
import type { AxiosResponse } from "axios";
import { useAuthStore } from "./authStore";
import type {
  AdminUser,
  ApiEnvelope,
  BloodStock,
  Donor,
  DonorCheckinRequest,
  EmergencyRequest,
  Hospital,
  LiveResponse,
} from "./types";

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "/api/v1",
  withCredentials: true,
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

const unwrap = <T>(response: AxiosResponse<ApiEnvelope<T>>) => response.data.data;

export function apiErrorMessage(error: unknown, fallback: string) {
  if (axios.isAxiosError<{ message?: string }>(error)) {
    return error.response?.data?.message ?? fallback;
  }
  return fallback;
}

export const endpoints = {
  login: async (username: string, password: string) =>
    unwrap(
      await api.post<
        ApiEnvelope<{
          token: string;
          user: AdminUser;
        }>
      >("/auth/admin/login", { username, password }),
    ),
  me: async () => unwrap(await api.get<ApiEnvelope<AdminUser>>("/auth/me")),
  stock: async () => unwrap(await api.get<ApiEnvelope<BloodStock[]>>("/stock")),
  updateStock: async (bloodType: string, productType: string, payload: unknown) =>
    unwrap(
      await api.put<ApiEnvelope<BloodStock>>(
        `/stock/${encodeURIComponent(bloodType)}/${encodeURIComponent(productType)}`,
        payload,
      ),
    ),
  requests: async () => unwrap(await api.get<ApiEnvelope<EmergencyRequest[]>>("/emergency/requests")),
  createRequest: async (payload: unknown) =>
    unwrap(await api.post<ApiEnvelope<EmergencyRequest>>("/emergency/requests", payload)),
  eligibleDonors: async (id: string) =>
    unwrap(
      await api.get<
        ApiEnvelope<{
          request: EmergencyRequest;
          donors: Donor[];
          count: number;
        }>
      >(`/emergency/requests/${id}/eligible-donors`),
    ),
  broadcast: async (id: string) =>
    unwrap(
      await api.post<
        ApiEnvelope<{
          broadcastId: string;
          recipientCount: number;
          queuedAt: string;
        }>
      >(`/emergency/requests/${id}/broadcast`),
    ),
  liveResponses: async (id: string) =>
    unwrap(await api.get<ApiEnvelope<LiveResponse[]>>(`/emergency/requests/${id}/live-responses`)),
  closeRequest: async (id: string) =>
    unwrap(await api.put<ApiEnvelope<EmergencyRequest>>(`/emergency/requests/${id}/close`)),
  donors: async (search = "") => unwrap(await api.get<ApiEnvelope<Donor[]>>("/donors", { params: { search } })),
  donor: async (uuid: string) => unwrap(await api.get<ApiEnvelope<Donor>>(`/donors/${encodeURIComponent(uuid)}`)),
  donorCheckinRequests: async (uuid: string) => {
    const data = unwrap(
      await api.get<ApiEnvelope<DonorCheckinRequest[] | null>>(`/donors/${encodeURIComponent(uuid)}/checkin-requests`),
    );
    return data ?? [];
  },
  createDonor: async (payload: unknown) => unwrap(await api.post<ApiEnvelope<Donor>>("/donors", payload)),
  updateDonor: async (id: string, payload: unknown) => unwrap(await api.put<ApiEnvelope<Donor>>(`/donors/${id}`, payload)),
  updateDonorStatus: async (id: string, isActive: boolean) =>
    unwrap(await api.put<ApiEnvelope<Donor>>(`/donors/${id}/status`, { isActive })),
  checkin: async (payload: unknown) =>
    unwrap(
      await api.post<
        ApiEnvelope<{
          donationId: string;
          isEligible: boolean;
          reasons: string[];
        }>
      >("/donations/checkin", payload),
    ),
  hospitals: async () => unwrap(await api.get<ApiEnvelope<Hospital[]>>("/hospitals")),
  createHospital: async (payload: unknown) => unwrap(await api.post<ApiEnvelope<Hospital>>("/hospitals", payload)),
  updateHospital: async (id: string, payload: unknown) =>
    unwrap(await api.put<ApiEnvelope<Hospital>>(`/hospitals/${id}`, payload)),
  deleteHospital: async (id: string) => unwrap(await api.delete<ApiEnvelope<unknown>>(`/hospitals/${id}`)),
};
