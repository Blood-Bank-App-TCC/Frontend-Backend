import { Siren } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodStock, EmergencyRequest } from "../../models/types";
import { bloodTypes, stockLevel } from "../../models/status";
import BloodTypeCard from "../components/BloodTypeCard";
import { RequestTable } from "../components/DataTables";
import Metric from "../components/Metric";
import PageHeader from "../layout/PageHeader";

export default function DashboardPage() {
  const [stock, setStock] = useState<BloodStock[]>([]);
  const [requests, setRequests] = useState<EmergencyRequest[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([bankDarahController.stock(), bankDarahController.requests()])
      .then(([stockData, requestData]) => {
        setStock(stockData);
        setRequests(requestData);
      })
      .catch((err) => setError(apiErrorMessage(err, "Dashboard gagal memuat data backend.")));
  }, []);

  const grouped = useMemo(
    () =>
      bloodTypes.map((bloodType) => ({
        bloodType,
        items: stock.filter((item) => item.bloodType === bloodType),
      })),
    [stock]
  );
  const critical = stock.filter((item) => stockLevel(item) === "critical").length;
  const low = stock.filter((item) => stockLevel(item) === "low").length;
  const activeRequests = requests.filter((request) => request.status === "ACTIVE" || request.status === "PENDING");

  return (
    <>
      <PageHeader
        eyebrow="Operasi stok real-time"
        title="Dashboard Stok Darah"
        action={
          <Link className="btn primary" to="/emergency/new">
            <Siren size={18} />
            Permintaan Baru
          </Link>
        }
      />
      {error && <p className="alert danger">{error}</p>}
      <section className="metric-grid">
        <Metric label="Total Kantong" value={stock.reduce((sum, item) => sum + item.quantity, 0)} tone="info" />
        <Metric label="Stok Kritis" value={critical} tone="danger" />
        <Metric label="Stok Menipis" value={low} tone="warning" />
        <Metric label="Request Aktif" value={activeRequests.length} tone="success" />
      </section>
      <section className="blood-grid">
        {grouped.map((group) => (
          <BloodTypeCard bloodType={group.bloodType} items={group.items} key={group.bloodType} />
        ))}
      </section>
      <section className="panel">
        <div className="panel-header">
          <h2>Permintaan Terbaru</h2>
          <Link to="/reports">Lihat laporan</Link>
        </div>
        <RequestTable requests={requests.slice(0, 5)} />
      </section>
    </>
  );
}
