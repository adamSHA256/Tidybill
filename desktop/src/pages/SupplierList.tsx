import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Badge,
  Button,
  Table,
  TextInput,
  Textarea,
  Select,
  Switch,
  Modal,
  Loader,
  Center,
  Avatar,
  FileButton,
  ActionIcon,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconUpload, IconTrash, IconPencil } from '@tabler/icons-react'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Supplier } from '../api/client'
import { useT } from '../i18n'

export function SupplierList() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editingSupplier, setEditingSupplier] = useState<Supplier | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Supplier | null>(null)

  // Form state
  const [name, setName] = useState('')
  const [ico, setIco] = useState('')
  const [dic, setDic] = useState('')
  const [street, setStreet] = useState('')
  const [city, setCity] = useState('')
  const [zip, setZip] = useState('')
  const [country, setCountry] = useState('CZ')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [website, setWebsite] = useState('')
  const [invoicePrefix, setInvoicePrefix] = useState('')
  const [isVatPayer, setIsVatPayer] = useState(false)
  const [notes, setNotes] = useState('')

  const queryClient = useQueryClient()
  const { t } = useT()

  const { data: suppliers, isLoading } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const uploadMutation = useMutation({
    mutationFn: ({ id, file }: { id: string; file: File }) => api.uploadLogo(id, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.logo_uploaded'), message: t('notify.logo_uploaded_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('notify.upload_failed'), message: err.message, color: 'red' })
    },
  })

  const deleteLogoMutation = useMutation({
    mutationFn: (id: string) => api.deleteLogo(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.logo_removed'), message: t('notify.logo_removed_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const saveMutation = useMutation({
    mutationFn: (data: Partial<Supplier>) =>
      editingSupplier ? api.updateSupplier(editingSupplier.id, data) : api.createSupplier(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({
        title: editingSupplier ? t('notify.supplier_updated') : t('notify.supplier_created'),
        message: t('notify.supplier_saved_msg').replace('{name}', name),
        color: 'green',
      })
      closeModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteSupplier(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.supplier_deleted'), message: t('notify.supplier_deleted_msg'), color: 'green' })
      setDeleteOpen(false)
      setDeleteTarget(null)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCreate = () => {
    setEditingSupplier(null)
    setName('')
    setIco('')
    setDic('')
    setStreet('')
    setCity('')
    setZip('')
    setCountry('CZ')
    setEmail('')
    setPhone('')
    setWebsite('')
    setInvoicePrefix('')
    setIsVatPayer(false)
    setNotes('')
    setModalOpen(true)
  }

  const openEdit = (supplier: Supplier) => {
    setEditingSupplier(supplier)
    setName(supplier.name)
    setIco(supplier.ico)
    setDic(supplier.dic)
    setStreet(supplier.street)
    setCity(supplier.city)
    setZip(supplier.zip)
    setCountry(supplier.country || 'CZ')
    setEmail(supplier.email)
    setPhone(supplier.phone)
    setWebsite(supplier.website)
    setInvoicePrefix(supplier.invoice_prefix)
    setIsVatPayer(supplier.is_vat_payer)
    setNotes(supplier.notes)
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingSupplier(null)
  }

  const handleSave = () => {
    if (!name.trim()) {
      notifications.show({ title: t('supplier.missing_name_title'), message: t('supplier.missing_name_msg'), color: 'orange' })
      return
    }
    saveMutation.mutate({
      name: name.trim(),
      ico: ico.trim(),
      dic: dic.trim(),
      street: street.trim(),
      city: city.trim(),
      zip: zip.trim(),
      country,
      email: email.trim(),
      phone: phone.trim(),
      website: website.trim(),
      invoice_prefix: invoicePrefix.trim(),
      is_vat_payer: isVatPayer,
      notes: notes.trim(),
    })
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('supplier.title')}</Title>
          <Text c="dimmed" size="sm">{t('supplier.subtitle')}</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>{t('supplier.add')}</Button>
      </Group>

      <Paper p="md" radius="md" withBorder>
        {(suppliers || []).length === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">{t('supplier.no_suppliers')}</Text>
        ) : (
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t('supplier.logo')}</Table.Th>
                <Table.Th>{t('supplier.name')}</Table.Th>
                <Table.Th>{t('supplier.ico')}</Table.Th>
                <Table.Th>{t('supplier.dic')}</Table.Th>
                <Table.Th>{t('supplier.address')}</Table.Th>
                <Table.Th>{t('supplier.vat_payer')}</Table.Th>
                <Table.Th></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {(suppliers || []).map((s) => (
                <Table.Tr key={s.id}>
                  <Table.Td>
                    <Group gap="xs">
                      {s.logo_path ? (
                        <Avatar
                          src={api.getLogoUrl(s.id)}
                          size={36}
                          radius="sm"
                        />
                      ) : (
                        <Avatar size={36} radius="sm" color="gray">
                          {s.name.charAt(0).toUpperCase()}
                        </Avatar>
                      )}
                      <Group gap={4}>
                        <FileButton
                          onChange={(file) => {
                            if (file) uploadMutation.mutate({ id: s.id, file })
                          }}
                          accept="image/png,image/jpeg"
                        >
                          {(props) => (
                            <ActionIcon variant="subtle" size="sm" color="blue" {...props}>
                              <IconUpload size={14} />
                            </ActionIcon>
                          )}
                        </FileButton>
                        {s.logo_path && (
                          <ActionIcon
                            variant="subtle"
                            size="sm"
                            color="red"
                            onClick={() => deleteLogoMutation.mutate(s.id)}
                          >
                            <IconTrash size={14} />
                          </ActionIcon>
                        )}
                      </Group>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Text size="sm" fw={500}>{s.name}</Text>
                      {s.is_default && <Badge size="xs" color="blue">{t('supplier.default')}</Badge>}
                    </Group>
                  </Table.Td>
                  <Table.Td fz="sm">{s.ico}</Table.Td>
                  <Table.Td fz="sm">{s.dic || '—'}</Table.Td>
                  <Table.Td fz="sm">{[s.street, s.city, s.zip].filter(Boolean).join(', ')}</Table.Td>
                  <Table.Td>
                    <Badge size="xs" color={s.is_vat_payer ? 'green' : 'gray'} variant="light">
                      {s.is_vat_payer ? t('supplier.yes') : t('supplier.no')}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(s)}>
                        <IconPencil size={14} />
                      </ActionIcon>
                      <ActionIcon variant="light" size="sm" color="red" onClick={() => { setDeleteTarget(s); setDeleteOpen(true) }}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Paper>

      <Modal opened={modalOpen} onClose={closeModal}
        title={editingSupplier ? t('supplier.edit_title') : t('supplier.new_title')} size="lg">
        <Stack gap="md">
          <TextInput label={t('supplier.name_label')} value={name}
            onChange={(e) => setName(e.currentTarget.value)} required />
          <Group grow>
            <TextInput label={t('supplier.ico_label')} value={ico}
              onChange={(e) => setIco(e.currentTarget.value)} />
            <TextInput label={t('supplier.dic_label')} value={dic}
              onChange={(e) => setDic(e.currentTarget.value)} />
          </Group>
          <TextInput label={t('supplier.street_label')} value={street}
            onChange={(e) => setStreet(e.currentTarget.value)} />
          <Group grow>
            <TextInput label={t('supplier.city_label')} value={city}
              onChange={(e) => setCity(e.currentTarget.value)} />
            <TextInput label={t('supplier.zip_label')} value={zip}
              onChange={(e) => setZip(e.currentTarget.value)} />
          </Group>
          <Select label={t('supplier.country_label')} data={['CZ', 'SK', 'DE', 'AT', 'PL', 'HU']}
            value={country} onChange={(v) => setCountry(v || 'CZ')} />
          <Group grow>
            <TextInput label={t('supplier.email_label')} value={email}
              onChange={(e) => setEmail(e.currentTarget.value)} />
            <TextInput label={t('supplier.phone_label')} value={phone}
              onChange={(e) => setPhone(e.currentTarget.value)} />
          </Group>
          <Group grow>
            <TextInput label={t('supplier.website_label')} value={website}
              onChange={(e) => setWebsite(e.currentTarget.value)} />
            <TextInput label={t('supplier.invoice_prefix_label')} value={invoicePrefix}
              onChange={(e) => setInvoicePrefix(e.currentTarget.value)} />
          </Group>
          <Switch label={t('supplier.is_vat_payer_label')} checked={isVatPayer}
            onChange={(e) => setIsVatPayer(e.currentTarget.checked)} />
          <Textarea label={t('supplier.notes_label')} value={notes}
            onChange={(e) => setNotes(e.currentTarget.value)} minRows={2} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeModal}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={saveMutation.isPending}>
              {editingSupplier ? t('common.save') : t('common.create')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      <Modal opened={deleteOpen} onClose={() => { setDeleteOpen(false); setDeleteTarget(null) }}
        title={t('supplier.delete_title')} size="sm">
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{
            __html: t('supplier.delete_confirm').replace('{name}', deleteTarget?.name || '')
          }} />
          <Group justify="end">
            <Button variant="default" onClick={() => { setDeleteOpen(false); setDeleteTarget(null) }}>{t('common.cancel')}</Button>
            <Button color="red" onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              loading={deleteMutation.isPending}>{t('common.delete')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
