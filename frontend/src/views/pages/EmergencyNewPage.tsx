import { Ambulance } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodType, ProductType, UrgencyLevel } from "../../models/types";
import { bloodTypes, productTypes } from "../../models/status";
import PageHeader from "../layout/PageHeader";

export default function EmergencyNewPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState({
    hospitalName: "RS Bethesda Yogyakarta",
    picName: "dr. Nadya",
    picPhone: "+628123450011",
    bloodType: "O-" as BloodType,
    productType: "PRC" as ProductType,
    quantityNeeded: 3,
    urgencyLevel: "CRITICAL" as UrgencyLevel,
    notes: "Perdarahan pascaoperasi, butuh donor secepatnya.",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const request = await bankDarahController.createRequest(form);
      navigate(`/emergency/${request.id}/broadcast`);
    } catch (err) {
      setError(apiErrorMessage(err, "Permintaan darurat gagal dibuat."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <PageHeader eyebrow="Emergency request" title="Form Permintaan Darurat" />
      {error && <p className="alert danger">{error}</p>}
      <form className="panel form-grid" onSubmit={submit}>
        <label>
          Nama Rumah Sakit
          <input value={form.hospitalName} onChange={(event) => setForm({ ...form, hospitalName: event.target.value })} required />
        </label>
        <label>
          Nama PIC
          <input value={form.picName} onChange={(event) => setForm({ ...form, picName: event.target.value })} required />
        </label>
        <label>
          Nomor Kontak
          <input value={form.picPhone} onChange={(event) => setForm({ ...form, picPhone: event.target.value })} required />
        </label>
        <label>
          Golongan Darah
          <select value={form.bloodType} onChange={(event) => setForm({ ...form, bloodType: event.target.value as BloodType })}>
            {bloodTypes.map((type) => (
              <option key={type}>{type}</option>
            ))}
          </select>
        </label>
        <label>
          Produk
          <select value={form.productType} onChange={(event) => setForm({ ...form, productType: event.target.value as ProductType })}>
            {productTypes.map((type) => (
              <option key={type}>{type}</option>
            ))}
          </select>
        </label>
        <label>
          Jumlah Kantong
          <input min={1} type="number" value={form.quantityNeeded} onChange={(event) => setForm({ ...form, quantityNeeded: Number(event.target.value) })} />
        </label>
        <label>
          Tingkat Urgensi
          <select value={form.urgencyLevel} onChange={(event) => setForm({ ...form, urgencyLevel: event.target.value as UrgencyLevel })}>
            <option>CRITICAL</option>
            <option>URGENT</option>
            <option>NORMAL</option>
          </select>
        </label>
        <label className="span-2">
          Catatan
          <textarea value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} />
        </label>
        <div className="form-actions span-2">
          <button className="btn primary" disabled={loading} type="submit">
            <Ambulance size={18} />
            {loading ? "Menyimpan..." : "Cari Donor Eligible"}
          </button>
        </div>
      </form>
    </>
  );
}
