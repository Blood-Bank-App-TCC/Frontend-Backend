export type BloodType = "A+" | "A-" | "B+" | "B-" | "O+" | "O-" | "AB+" | "AB-";
export type ProductType = "WB" | "PRC" | "FFP" | "THROMBOCYTE";
export type UrgencyLevel = "CRITICAL" | "URGENT" | "NORMAL";
export type ResponseStatus = "ACCEPTED" | "ON_THE_WAY" | "DECLINED" | "CHECKED_IN" | "NO_RESPONSE";
export type RequestStatus = "PENDING" | "ACTIVE" | "FULFILLED" | "EXPIRED" | "CLOSED";

export interface AdminUser {
  id: string;
  username: string;
  fullName: string;
  role: "SUPER_ADMIN" | "OPERATOR";
}

export interface ApiEnvelope<T> {
  success: boolean;
  data: T;
  message?: string;
  code?: string;
}

export interface BloodStock {
  id: string;
  bloodType: BloodType;
  productType: ProductType;
  quantity: number;
  safeThreshold: number;
  criticalThreshold: number;
  description: string;
  updatedAt: string;
}

export interface EmergencyRequest {
  id: string;
  hospitalName: string;
  picName: string;
  picPhone: string;
  bloodType: BloodType;
  productType: ProductType;
  quantityNeeded: number;
  urgencyLevel: UrgencyLevel;
  notes: string;
  status: RequestStatus;
  broadcastId?: string;
  eligibleCount: number;
  createdAt: string;
  broadcastSentAt?: string;
  fulfilledAt?: string;
}

export interface DonorCheckinRequest extends EmergencyRequest {
  responseStatus: ResponseStatus;
  respondedAt: string;
}

export interface DonationRecord {
  id: string;
  date: string;
  location: string;
  requestId?: string;
  bloodPressure: string;
  hemoglobin: number;
  weight: number;
  status: string;
}

export interface Donor {
  id: string;
  uuid: string;
  fullName: string;
  phone: string;
  email?: string;
  bloodType: BloodType;
  gender: "M" | "F";
  address: string;
  distanceKm: number;
  lastDonation?: string;
  nextEligible?: string;
  isEligible: boolean;
  isActive: boolean;
  donationHistory?: DonationRecord[];
}

export interface Hospital {
  id: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  picName: string;
  picPhone: string;
  email?: string;
  isActive: boolean;
}

export interface LiveResponse {
  id: string;
  broadcastId: string;
  donorId: string;
  donorName: string;
  bloodType: BloodType;
  distanceKm: number;
  status: ResponseStatus;
  respondedAt: string;
}
