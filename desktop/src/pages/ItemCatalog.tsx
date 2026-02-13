import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Table,
  Badge,
  Button,
  TextInput,
  NumberInput,
  Select,
  Modal,
  ActionIcon,
  Loader,
  Center,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconSearch, IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, type Item } from '../api/client'
import { useT } from '../i18n'

export function ItemCatalog() {
  const [search, setSearch] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<Item | null>(null)

  // Form state
  const [description, setDescription] = useState('')
  const [defaultPrice, setDefaultPrice] = useState<number>(0)
  const [defaultUnit, setDefaultUnit] = useState('ks')
  const [defaultVATRate, setDefaultVATRate] = useState<number>(21)
  const [category, setCategory] = useState('')

  const queryClient = useQueryClient()
  const { t } = useT()

  const { data: items, isLoading } = useQuery({
    queryKey: ['items', search],
    queryFn: () => api.getItems(search || undefined),
  })

  const { data: categories } = useQuery({
    queryKey: ['item-categories'],
    queryFn: api.getItemCategories,
  })

  const createMutation = useMutation({
    mutationFn: (data: Partial<Item>) =>
      editingItem ? api.updateItem(editingItem.id, data) : api.createItem(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['items'] })
      queryClient.invalidateQueries({ queryKey: ['item-categories'] })
      notifications.show({
        title: editingItem ? t('notify.item_updated') : t('notify.item_created'),
        message: t('notify.item_saved_msg').replace('{description}', description),
        color: 'green',
      })
      closeModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: api.deleteItem,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['items'] })
      notifications.show({ title: t('notify.item_deleted'), message: t('notify.item_deleted_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCreate = () => {
    setEditingItem(null)
    setDescription('')
    setDefaultPrice(0)
    setDefaultUnit('ks')
    setDefaultVATRate(21)
    setCategory('')
    setModalOpen(true)
  }

  const openEdit = (item: Item) => {
    setEditingItem(item)
    setDescription(item.description)
    setDefaultPrice(item.default_price)
    setDefaultUnit(item.default_unit)
    setDefaultVATRate(item.default_vat_rate)
    setCategory(item.category)
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingItem(null)
  }

  const handleSave = () => {
    if (!description.trim()) {
      notifications.show({ title: t('item.missing_desc_title'), message: t('item.missing_desc_msg'), color: 'orange' })
      return
    }
    createMutation.mutate({
      description: description.trim(),
      default_price: defaultPrice,
      default_unit: defaultUnit,
      default_vat_rate: defaultVATRate,
      category: category.trim(),
    })
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const categoryOptions = (categories || []).map((c) => ({ value: c, label: c }))

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('item.title')}</Title>
          <Text c="dimmed" size="sm">{t('item.subtitle')}</Text>
        </div>
        <Group>
          <TextInput
            placeholder={t('item.search')}
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
          />
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>{t('item.add')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        {(items || []).length === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">
            {search ? t('item.no_match') : t('item.no_items')}
          </Text>
        ) : (
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t('item.description')}</Table.Th>
                <Table.Th>{t('item.category')}</Table.Th>
                <Table.Th>{t('item.price')}</Table.Th>
                <Table.Th>{t('item.unit')}</Table.Th>
                <Table.Th>{t('item.vat')}</Table.Th>
                <Table.Th>{t('item.used')}</Table.Th>
                <Table.Th></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {(items || []).map((item) => (
                <Table.Tr key={item.id}>
                  <Table.Td>
                    <Text size="sm" fw={500}>{item.description}</Text>
                  </Table.Td>
                  <Table.Td>
                    {item.category ? (
                      <Badge size="xs" variant="light">{item.category}</Badge>
                    ) : (
                      <Text size="sm" c="dimmed">—</Text>
                    )}
                  </Table.Td>
                  <Table.Td fz="sm">{formatMoney(item.default_price)}</Table.Td>
                  <Table.Td fz="sm">{item.default_unit}</Table.Td>
                  <Table.Td fz="sm">{item.default_vat_rate}%</Table.Td>
                  <Table.Td>
                    <Badge size="xs" variant="light" color={item.usage_count > 0 ? 'blue' : 'gray'}>
                      {item.usage_count}x
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(item)}>
                        <IconPencil size={14} />
                      </ActionIcon>
                      <ActionIcon variant="light" size="sm" color="red"
                        onClick={() => deleteMutation.mutate(item.id)}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}

        {(items || []).length > 0 && (
          <Group justify="space-between" mt="md">
            <Text size="sm" c="dimmed">{t('item.showing').replace('{count}', String((items || []).length))}</Text>
          </Group>
        )}
      </Paper>

      <Modal opened={modalOpen} onClose={closeModal}
        title={editingItem ? t('item.edit_title') : t('item.new_title')} size="md">
        <Stack gap="md">
          <TextInput label={t('item.description_label')} placeholder={t('item.description_placeholder')}
            value={description} onChange={(e) => setDescription(e.currentTarget.value)} required />
          <Group grow>
            <NumberInput label={t('item.default_price')} min={0} value={defaultPrice}
              onChange={(val) => setDefaultPrice(Number(val) || 0)} />
            <Select label={t('item.unit_label')} data={['ks', 'hod', 'den', 'm\u00B2']}
              value={defaultUnit} onChange={(v) => setDefaultUnit(v || 'ks')} />
          </Group>
          <Group grow>
            <Select label={t('item.vat_rate')} data={['0', '12', '21']}
              value={String(defaultVATRate)}
              onChange={(v) => setDefaultVATRate(Number(v) || 0)} />
            <TextInput label={t('item.category_label')} placeholder={t('item.category_placeholder')}
              value={category} onChange={(e) => setCategory(e.currentTarget.value)}
              description={categoryOptions.length > 0 ? t('item.existing_categories').replace('{categories}', (categories || []).join(', ')) : undefined} />
          </Group>
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeModal}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={createMutation.isPending}>
              {editingItem ? t('common.save') : t('common.create')}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
