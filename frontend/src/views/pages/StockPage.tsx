import { type FormEvent, useEffect, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodStock, BloodType, ProductType } from "../../models/types";
import { bloodTypes, productTypes } from "../../models/status";
import { StockTable } from "../components/DataTables";
import PageHeader from "../layout/PageHeader";

export default function StockPage() {
  const [stock, setStock] = useState<BloodStock[]>([]);
  const [form, setForm] = useState({ bloodType: "O-" as BloodType, productType: "PRC" as ProductType, mode: "add", quantity: 1, reference: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function load() {
    setLoading(true);
    try {
      setStock(await bankDarahController.stock());
    } catch (err) {
      setError(apiErrorMessage(err, "Stok darah gagal dimuat."));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await bankDarahController.updateStock(form.bloodType, form.productType, {
        mode: form.mode,
        quantity: form.quantity,
        reference: form.reference,
      });
      setMessage("Stok darah berhasil diperbarui.");
      await load();
    } catch (err) {
      setError(apiErrorMessage(err, "Stok darah gagal diperbarui."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <PageHeader eyebrow="Transaksi stok" title="Input & Update Stok Darah" />
      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}
      <form className="panel compact-form" onSubmit={submit}>
        <select value={form.bloodType} onChange={(event) => setForm({ ...form, bloodType: event.target.value as BloodType })}>
          {bloodTypes.map((type) => (
            <option key={type}>{type}</option>
          ))}
        </select>
        <select value={form.productType} onChange={(event) => setForm({ ...form, productType: event.target.value as ProductType })}>
          {productTypes.map((type) => (
            <option key={type}>{type}</option>
          ))}
        </select>
        <select value={form.mode} onChange={(event) => setForm({ ...form, mode: event.target.value })}>
          <option value="add">Tambah</option>
          <option value="subtract">Kurangi</option>
          <option value="set">Set jumlah</option>
        </select>
        <input min={0} required type="number" value={form.quantity} onChange={(event) => setForm({ ...form, quantity: Number(event.target.value) })} />
        <input value={form.reference} onChange={(event) => setForm({ ...form, reference: event.target.value })} placeholder="Referensi" />
        <button className="btn primary" type="submit" disabled={saving}>
          {saving ? "Menyimpan..." : "Simpan"}
        </button>
      </form>
      <section className="panel">
        {loading ? <p className="subtle">Memuat stok darah...</p> : <StockTable stock={stock} />}
      </section>
    </>
  );
}
