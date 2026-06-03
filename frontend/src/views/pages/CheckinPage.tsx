import { ShieldCheck } from "lucide-react";
import { type FormEvent, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { Donor, DonorCheckinRequest } from "../../models/types";
import { formatDate, responseLabel } from "../../models/status";
import QRScanner from "../components/QRScanner";
import StatusBadge from "../components/StatusBadge";
import PageHeader from "../layout/PageHeader";

export default function CheckinPage() {
  const [donor, setDonor] = useState<Donor | null>(null);
  const [requests, setRequests] = useState<DonorCheckinRequest[]>([]);
  const [loadingRequests, setLoadingRequests] = useState(false);
  const [scanError, setScanError] = useState("");
  const [form, setForm] = useState({ systolic: 122, diastolic: 80, hemoglobin: 13.4, weight: 62, requestId: "" });
  const [result, setResult] = useState<{ isEligible: boolean; reasons: string[] } | null>(null);
  const [saving, setSaving] = useState(false);
  const [submitError, setSubmitError] = useState("");

  async function handleScan(value: string) {
    setScanError("");
    setResult(null);
    setSubmitError("");
    setDonor(null);
    setRequests([]);
    setLoadingRequests(false);
    setForm((current) => ({ ...current, requestId: "" }));
    try {
      const data = await bankDarahController.donor(value);
      setDonor(data);
      setLoadingRequests(true);
      try {
        const eligibleRequests = await bankDarahController.donorCheckinRequests(data.uuid);
        setRequests(eligibleRequests);
        setForm((current) => ({ ...current, requestId: eligibleRequests[0]?.id ?? "" }));
        if (eligibleRequests.length === 0) {
          setSubmitError("Donor ini belum memilih ingin donor pada request aktif mana pun.");
        }
      } catch (err) {
        setSubmitError(apiErrorMessage(err, "Request check-in donor gagal dimuat."));
      } finally {
        setLoadingRequests(false);
      }
    } catch {
      setDonor(null);
      setLoadingRequests(false);
      setScanError("Token QR tidak ditemukan atau sudah kedaluwarsa.");
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!donor) return;
    setSaving(true);
    setSubmitError("");
    setResult(null);
    if (!form.requestId) {
      setSubmitError("Pilih request aktif terlebih dahulu.");
      setSaving(false);
      return;
    }
    try {
      const data = await bankDarahController.checkin({ donorUuid: donor.uuid, ...form });
      setResult(data);
      setDonor(await bankDarahController.donor(donor.uuid));
    } catch (err) {
      setSubmitError(apiErrorMessage(err, "Check-in donor gagal diproses."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <PageHeader eyebrow="Verifikasi fisik donor" title="QR Check-in Donor" />
      <div className="split-grid">
        <QRScanner onScan={handleScan} />
        <section className="panel">
          <h2>Profil Donor</h2>
          {scanError && <p className="alert danger">{scanError}</p>}
          {donor ? (
            <div className="profile-block">
              <div className="avatar">{donor.bloodType}</div>
              <div>
                <h3>{donor.fullName}</h3>
                <p>{donor.phone}</p>
                <p>{donor.address}</p>
              </div>
              <StatusBadge variant={donor.isEligible ? "safe" : "critical"}>{donor.isEligible ? "Eligible" : "Belum eligible"}</StatusBadge>
            </div>
          ) : (
            <p className="empty">Scan QR untuk mengambil profil pendonor.</p>
          )}
          {donor && (
            <div className="history-list">
              {donor.donationHistory?.slice(0, 3).map((record) => (
                <div key={record.id}>
                  <strong>{formatDate(record.date)}</strong>
                  <span>{record.location}</span>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
      {donor && (
        <form className="panel form-grid" onSubmit={submit}>
          <label>
            Sistolik
            <input type="number" value={form.systolic} onChange={(event) => setForm({ ...form, systolic: Number(event.target.value) })} />
          </label>
          <label>
            Diastolik
            <input type="number" value={form.diastolic} onChange={(event) => setForm({ ...form, diastolic: Number(event.target.value) })} />
          </label>
          <label>
            Hemoglobin
            <input
              step="0.1"
              type="number"
              value={form.hemoglobin}
              onChange={(event) => setForm({ ...form, hemoglobin: Number(event.target.value) })}
            />
          </label>
          <label>
            Berat Badan
            <input type="number" value={form.weight} onChange={(event) => setForm({ ...form, weight: Number(event.target.value) })} />
          </label>
          <label className="span-2">
            Request Aktif
            <select
              value={form.requestId}
              onChange={(event) => setForm({ ...form, requestId: event.target.value })}
              required
              disabled={loadingRequests || requests.length === 0}
            >
              {requests.length === 0 && (
                <option value="">{loadingRequests ? "Memuat request donor..." : "Tidak ada request aktif untuk donor ini"}</option>
              )}
              {requests.map((request) => (
                <option key={request.id} value={request.id}>
                  {request.hospitalName} - {request.bloodType}/{request.productType} - {responseLabel(request.responseStatus)} -{" "}
                  {request.quantityNeeded} kantong
                </option>
              ))}
            </select>
          </label>
          <div className="form-actions span-2">
            <button className="btn primary" type="submit" disabled={saving || loadingRequests || requests.length === 0}>
              <ShieldCheck size={18} />
              {saving ? "Memproses..." : "Konfirmasi Check-in"}
            </button>
          </div>
          {submitError && <div className="alert danger span-2">{submitError}</div>}
          {result && (
            <div className={`alert ${result.isEligible ? "success" : "danger"} span-2`}>
              {result.isEligible ? "QR donor berhasil discan. Status donor selesai." : result.reasons.join(", ")}
            </div>
          )}
        </form>
      )}
    </>
  );
}
