/*
   This file is part of Astarte.

   Copyright 2021-2022 Ispirata Srl

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import React, { ChangeEvent, useState, useEffect } from 'react';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './DataSource';
import { AppEngineDataSourceOptions, AppEngineQuery } from './types';

const { FormField } = LegacyForms;

type Props = QueryEditorProps<DataSource, AppEngineQuery, AppEngineDataSourceOptions>;

function isValidQuery(query: AppEngineQuery) {
  return query.interfaceName !== '' && query.device !== '';
}

export const QueryEditor = ({ datasource, query, onChange, onRunQuery }: Props) => {
  const [interfaces, setInterfaces] = useState<string[]>([]);
  const { device, interfaceName, path } = query;

  useEffect(() => {
    datasource
      .getResource('interfaces', { device_id: query.device })
      .then(setInterfaces)
      .catch((error) => {
        console.error(error);
        setInterfaces([]);
      });
  }, [datasource, query.device]);

  const onDeviceChange = (event: ChangeEvent<HTMLInputElement>) => {
    const deviceId = event.target.value;
    const updatedQuery = { ...query, device: deviceId };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  const onInterfaceNameChange = (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const updatedQuery = { ...query, interfaceName: event.target.value };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  const onPathChange = (event: ChangeEvent<HTMLInputElement>) => {
    const updatedQuery = { ...query, path: event.target.value };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  return (
    <div className="gf-form">
      <FormField width={4} value={device} onChange={onDeviceChange} label="Device ID" tooltip="The device ID" />
      {interfaces.length > 0 ? (
        <FormField
          label="Interface"
          labelWidth={4}
          inputEl={
            <select className="gf-form-input width-20" value={interfaceName} onChange={onInterfaceNameChange}>
              <option value="">Select an interface</option>
              {interfaces.map((iface) => (
                <option key={iface} value={iface}>
                  {iface}
                </option>
              ))}
            </select>
          }
        />
      ) : (
        <FormField
          labelWidth={4}
          value={interfaceName}
          onChange={onInterfaceNameChange}
          label="Interface"
          tooltip="The interface to query"
        />
      )}
      <FormField width={4} value={path} onChange={onPathChange} label="Path" tooltip="The interface path to query" />
    </div>
  );
};
