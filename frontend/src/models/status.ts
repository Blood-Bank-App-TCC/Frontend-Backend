import type { BloodStock, ResponseStatus, UrgencyLevel } from "./types";

export const bloodTypes = ["A+", "A-", "B+", "B-", "O+", "O-", "AB+", "AB-"] as const;
export const productTypes = ["WB", "PRC", "FFP", "THROMBOCYTE"] as const;

export function stockLevel(item: BloodStock) {
  if (item.quantity <= item.criticalThreshold) return "critical";
  if (item.quantity < item.safeThreshold) return "low";
  return "safe";
}

export function stockLabel(item: BloodStock) {
  const level = stockLevel(item);
  if (level === "critical") return "Kritis";
  if (level === "low") return "Menipis";
  return "Aman";
}

export function urgencyLabel(level: UrgencyLevel) {
  return level === "CRITICAL" ? "Critical" : level === "URGENT" ? "Urgent" : "Normal";
}

export function responseLabel(status: ResponseStatus) {
  const labels: Record<ResponseStatus, string> = {
    ACCEPTED: "Siap donor",
    ON_THE_WAY: "Menuju PMI",
    DECLINED: "Tidak bisa",
    CHECKED_IN: "Selesai",
    NO_RESPONSE: "Belum respons",
  };
  return labels[status];
}

export function formatDateTime(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

export function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(new Date(value));
}
