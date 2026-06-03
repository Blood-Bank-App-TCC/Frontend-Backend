import { Download } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodStock, Donor, EmergencyRequest } from "../../models/types";
import { stockLevel } from "../../models/status";
import { RequestTable, StockTable } from "../components/DataTables";
import Metric from "../components/Metric";
import PageHeader from "../layout/PageHeader";

export default function ReportsPage() {
  const [stock, setStock] = useState<BloodStock[]>([]);
  const [requests, setRequests] = useState<EmergencyRequest[]>([]);
  const [donors, setDonors] = useState<Donor[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([bankDarahController.stock(), bankDarahController.requests(), bankDarahController.donors()])
      .then(([stockData, requestData, donorData]) => {
        setStock(stockData);
        setRequests(requestData);
        setDonors(donorData);
      })
      .catch((err) => setError(apiErrorMessage(err, "Data laporan gagal dimuat.")));
  }, []);

  const summary = useMemo(
    () => ({
      totalStock: stock.reduce((sum, item) => sum + item.quantity, 0),
      criticalStock: stock.filter((item) => stockLevel(item) === "critical").length,
      activeRequests: requests.filter((request) => request.status === "ACTIVE" || request.status === "PENDING").length,
      eligibleDonors: donors.filter((donor) => donor.isEligible && donor.isActive).length,
    }),
    [donors, requests, stock],
  );

  function exportCsv() {
    const rows = [
      ["section", "field_1", "field_2", "field_3", "field_4", "field_5", "field_6"],
      ["summary", "total_stock", summary.totalStock, "critical_stock", summary.criticalStock, "active_requests", summary.activeRequests],
      ["summary", "eligible_donors", summary.eligibleDonors, "", "", "", ""],
      ...requests.map((request) => [
        "request",
        request.hospitalName,
        `${request.bloodType}/${request.productType}`,
        request.quantityNeeded,
        request.urgencyLevel,
        request.status,
        request.createdAt,
      ]),
      ...stock.map((item) => [
        "stock",
        item.bloodType,
        item.productType,
        item.quantity,
        item.safeThreshold,
        item.criticalThreshold,
        item.updatedAt,
      ]),
      ...donors.map((donor) => [
        "donor",
        donor.fullName,
        donor.bloodType,
        donor.phone,
        donor.isEligible ? "eligible" : "not_eligible",
        donor.isActive ? "active" : "inactive",
        donor.nextEligible ?? "",
      ]),
    ];
    const csv = rows.map((row) => row.map(csvCell).join(",")).join("\r\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `laporan-bank-darah-${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <>
      <PageHeader
        eyebrow="Rekap operasional"
        title="Laporan Bank Darah"
        action={
          <button className="btn secondary" type="button" onClick={exportCsv}>
            <Download size={18} />
            Export CSV
          </button>
        }
      />
      {error && <p className="alert danger">{error}</p>}
      <section className="metric-grid">
        <Metric label="Total Kantong" value={summary.totalStock} tone="info" />
        <Metric label="Stok Kritis" value={summary.criticalStock} tone="danger" />
        <Metric label="Request Aktif" value={summary.activeRequests} tone="warning" />
        <Metric label="Donor Eligible" value={summary.eligibleDonors} tone="success" />
      </section>
      <section className="panel">
        <div className="panel-header">
          <h2>Permintaan Darurat</h2>
          <span className="subtle">{requests.length} permintaan</span>
        </div>
        <RequestTable requests={requests} />
      </section>
      <section className="panel">
        <div className="panel-header">
          <h2>Ringkasan Stok</h2>
          <span className="subtle">{stock.length} kombinasi darah dan produk</span>
        </div>
        <StockTable stock={stock} />
      </section>
    </>
  );
}

function csvCell(value: string | number) {
  const text = String(value);
  return `"${text.replace(/"/g, '""')}"`;
}
