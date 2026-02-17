import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  SimpleGrid,
  Avatar,
  Button,
  TextInput,
  NumberInput,
  Textarea,
  Modal,
  Divider,
  ActionIcon,
  Loader,
  Center,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconSearch, IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Customer } from '../api/client'
import { CountrySelect } from '../components/CountrySelect'
import { useT } from '../i18n'

const avatarColors = ['blue', 'green', 'yellow', 'violet', 'orange', 'teal', 'red', 'pink']

export function CustomerList() {
  const [search, setSearch] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingCustomer, setEditingCustomer] = useState<Customer | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Customer | null>(null)

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
  const [defaultDueDays, setDefaultDueDays] = useState<number>(0)
  const [notes, setNotes] = useState('')

  const queryClient = useQueryClient()
  const { t } = useT()

  const { data: customers, isLoading } = useQuery({
    queryKey: ['customers'],
    queryFn: () => api.getCustomers(),
  })

  const createMutation = useMutation({
    mutationFn: (data: Partial<Customer>) =>
      editingCustomer ? api.updateCustomer(editingCustomer.id, data) : api.createCustomer(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
      notifications.show({
        title: editingCustomer ? t('notify.customer_updated') : t('notify.customer_created'),
        message: t('notify.customer_saved_msg').replace('{name}', name),
        color: 'green',
      })
      closeModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteCustomer(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
      notifications.show({ title: t('notify.customer_deleted'), message: t('notify.customer_deleted_msg'), color: 'green' })
      setDeleteOpen(false)
      setDeleteTarget(null)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCreate = () => {
    setEditingCustomer(null)
    setName('')
    setIco('')
    setDic('')
    setStreet('')
    setCity('')
    setZip('')
    setCountry('CZ')
    setEmail('')
    setPhone('')
    setDefaultDueDays(0)
    setNotes('')
    setModalOpen(true)
  }

  const openEdit = (customer: Customer) => {
    setEditingCustomer(customer)
    setName(customer.name)
    setIco(customer.ico)
    setDic(customer.dic)
    setStreet(customer.street)
    setCity(customer.city)
    setZip(customer.zip)
    setCountry(customer.country || 'CZ')
    setEmail(customer.email)
    setPhone(customer.phone)
    setDefaultDueDays(customer.default_due_days || 0)
    setNotes(customer.notes)
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingCustomer(null)
  }

  const handleSave = () => {
    if (!name.trim()) {
      notifications.show({ title: t('customer.missing_name_title'), message: t('customer.missing_name_msg'), color: 'orange' })
      return
    }
    createMutation.mutate({
      name: name.trim(),
      ico: ico.trim(),
      dic: dic.trim(),
      street: street.trim(),
      city: city.trim(),
      zip: zip.trim(),
      country,
      email: email.trim(),
      phone: phone.trim(),
      default_due_days: defaultDueDays,
      notes: notes.trim(),
    })
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const filtered = (customers || []).filter(
    (c: Customer) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.ico.includes(search)
  )

  const initials = (n: string) =>
    n
      .split(/[\s.]+/)
      .filter((w) => w.length > 1)
      .slice(0, 2)
      .map((w) => w[0].toUpperCase())
      .join('')

  const colorForIndex = (i: number) => avatarColors[i % avatarColors.length]

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('customer.title')}</Title>
          <Text c="dimmed" size="sm">{t('customer.subtitle')}</Text>
        </div>
        <Group>
          <TextInput
            placeholder={t('customer.search')}
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
          />
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>{t('customer.add')}</Button>
        </Group>
      </Group>

      {filtered.length === 0 && !search ? (
        <Paper p="xl" radius="md" withBorder style={{ borderStyle: 'dashed', cursor: 'pointer' }} onClick={openCreate}>
          <Stack align="center" gap="xs">
            <IconPlus size={32} color="gray" />
            <Text c="dimmed" size="sm">{t('customer.no_customers')}</Text>
          </Stack>
        </Paper>
      ) : (
        <SimpleGrid cols={{ base: 1, md: 2 }}>
          {filtered.map((c, i) => (
            <Paper key={c.id} p="md" radius="md" withBorder>
              <Group>
                <Avatar color={colorForIndex(i)} radius="xl" size="lg">
                  {initials(c.name)}
                </Avatar>
                <div style={{ flex: 1 }}>
                  <Text fw={500}>{c.name}</Text>
                  <Text size="sm" c="dimmed">
                    ICO: {c.ico}{c.dic && ` | DIC: ${c.dic}`}
                  </Text>
                  <Text size="sm" c="dimmed">
                    {[c.street, c.city, c.zip].filter(Boolean).join(', ')}
                  </Text>
                  {c.email && <Text size="sm" c="dimmed">{c.email}</Text>}
                </div>
                <Group gap="xs">
                  <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(c)}>
                    <IconPencil size={14} />
                  </ActionIcon>
                  <ActionIcon variant="light" size="sm" color="red" onClick={() => { setDeleteTarget(c); setDeleteOpen(true) }}>
                    <IconTrash size={14} />
                  </ActionIcon>
                </Group>
              </Group>
              <Divider my="sm" />
              <Group>
                <Text size="xs" c="dimmed">
                  {t('customer.country')} <Text span fw={600} c="dark" size="xs">{c.country || 'CZ'}</Text>
                </Text>
                {c.phone && (
                  <Text size="xs" c="dimmed">
                    {t('customer.phone')} <Text span fw={600} c="dark" size="xs">{c.phone}</Text>
                  </Text>
                )}
              </Group>
            </Paper>
          ))}

          <Paper
            p="xl"
            radius="md"
            withBorder
            style={{ borderStyle: 'dashed', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 160 }}
            onClick={openCreate}
          >
            <Stack align="center" gap="xs">
              <IconPlus size={32} color="gray" />
              <Text c="dimmed" size="sm">{t('customer.add_new')}</Text>
            </Stack>
          </Paper>
        </SimpleGrid>
      )}

      <Modal opened={modalOpen} onClose={closeModal}
        title={editingCustomer ? t('customer.edit_title') : t('customer.new_title')} size="lg">
        <Stack gap="md">
          <TextInput label={t('customer.name_label')} value={name}
            onChange={(e) => setName(e.currentTarget.value)} required />
          <Group grow>
            <TextInput label={t('customer.ico_label')} value={ico}
              onChange={(e) => setIco(e.currentTarget.value)} />
            <TextInput label={t('customer.dic_label')} value={dic}
              onChange={(e) => setDic(e.currentTarget.value)} />
          </Group>
          <TextInput label={t('customer.street_label')} value={street}
            onChange={(e) => setStreet(e.currentTarget.value)} />
          <Group grow>
            <TextInput label={t('customer.city_label')} value={city}
              onChange={(e) => setCity(e.currentTarget.value)} />
            <TextInput label={t('customer.zip_label')} value={zip}
              onChange={(e) => setZip(e.currentTarget.value)} />
          </Group>
          <CountrySelect label={t('customer.country_label')}
            value={country} onChange={(v) => setCountry(v || 'CZ')} />
          <Group grow>
            <TextInput label={t('customer.email_label')} value={email}
              onChange={(e) => setEmail(e.currentTarget.value)} />
            <TextInput label={t('customer.phone_label')} value={phone}
              onChange={(e) => setPhone(e.currentTarget.value)} />
          </Group>
          <NumberInput label={t('customer.default_due_days_label')}
            description={t('customer.default_due_days_desc')}
            value={defaultDueDays} onChange={(v) => setDefaultDueDays(Number(v) || 0)}
            min={0} max={365} w={200} />
          <Textarea label={t('customer.notes_label')} value={notes}
            onChange={(e) => setNotes(e.currentTarget.value)} minRows={2} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeModal}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={createMutation.isPending}>
              {editingCustomer ? t('common.save') : t('common.create')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      <Modal opened={deleteOpen} onClose={() => { setDeleteOpen(false); setDeleteTarget(null) }}
        title={t('customer.delete_title')} size="sm">
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{
            __html: t('customer.delete_confirm').replace('{name}', deleteTarget?.name || '')
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
