import { Siren } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { Donor, EmergencyRequest } from "../../models/types";
import { DonorTable } from "../components/DataTables";
import Loading from "../components/Loading";
import StatusBadge from "../components/StatusBadge";
import PageHeader from "../layout/PageHeader";

export default function BroadcastPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [request, setRequest] = useState<EmergencyRequest | null>(null);
  const [donors, setDonors] = useState<Donor[]>([]);
  const [error, setError] = useState("");
  const [sending, setSending] = useState(false);

  useEffect(() => {
    bankDarahController
      .eligibleDonors(id)
      .then((data) => {
        setRequest(data.request);
        setDonors(data.donors);
      })
      .catch((err) => setError(apiErrorMessage(err, "Donor eligible gagal dimuat.")));
  }, [id]);

  async function sendBroadcast() {
    setError("");
    setSending(true);
    try {
      await bankDarahController.broadcast(id);
      navigate("/dashboard#permintaan-aktif", { state: { broadcastSuccess: true } });
    } catch (err) {
      setError(apiErrorMessage(err, "Broadcast gagal dikirim."));
    } finally {
      setSending(false);
    }
  }

  if (!request && error) return <p className="alert danger">{error}</p>;
  if (!request) return <Loading title="Menyiapkan donor eligible" />;

  return (
    <>
      <PageHeader
        eyebrow="Review donor eligible"
        title={`${request.bloodType} ${request.productType} untuk ${request.hospitalName}`}
        action={
          <button className="btn danger" disabled={sending} onClick={sendBroadcast} type="button">
            <Siren size={18} />
            {sending ? "Mengantrekan..." : "Kirim Broadcast"}
          </button>
        }
      />
      <section className={`emergency-summary ${request.urgencyLevel === "CRITICAL" ? "pulse" : ""}`}>
        <div>
          <span>Urgensi</span>
          <StatusBadge urgency={request.urgencyLevel} />
        </div>
        <div>
          <span>Kebutuhan</span>
          <strong>{request.quantityNeeded} kantong</strong>
        </div>
        <div>
          <span>Donor eligible</span>
          <strong>{donors.length} orang</strong>
        </div>
        <div>
          <span>PIC RS</span>
          <strong>{request.picName}</strong>
        </div>
      </section>
      {error && <p className="alert danger">{error}</p>}
      <section className="panel">
        <div className="panel-header">
          <h2>Daftar Donor Terdekat</h2>
          <span className="subtle">Radius simulasi 10 KM</span>
        </div>
        <DonorTable donors={donors} />
      </section>
    </>
  );
}
