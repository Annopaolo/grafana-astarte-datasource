/*
   This file is part of Astarte.

   Copyright 2021 Ispirata Srl

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

import defaults from 'lodash/defaults';

import React, { ChangeEvent, Component } from 'react';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './DataSource';
import { defaultQuery, AppEngineDataSourceOptions, AppEngineQuery } from './types';

const { FormField } = LegacyForms;

type Props = QueryEditorProps<DataSource, AppEngineQuery, AppEngineDataSourceOptions>;

interface State {}

function isValidQuery(query: AppEngineQuery) {
  return query.interfaceName !== '' && query.device !== '';
}

export class QueryEditor extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
  }

  onDeviceChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    const deviceId = event.target.value;
    const updatedQuery = { ...query, device: deviceId };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  onInterfaceNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    const updatedQuery = { ...query, interfaceName: event.target.value };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  onPathChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    const updatedQuery = { ...query, path: event.target.value };
    onChange(updatedQuery);
    if (isValidQuery(updatedQuery)) {
      onRunQuery();
    }
  };

  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { device, interfaceName, path } = query;

    return (
      <div className="gf-form">
        <FormField width={4} value={device} onChange={this.onDeviceChange} label="Device ID" tooltip="The device ID" />
        <FormField
          labelWidth={4}
          value={interfaceName}
          onChange={this.onInterfaceNameChange}
          label="Interface"
          tooltip="The interface to query"
        />
        <FormField
          width={4}
          value={path}
          onChange={this.onPathChange}
          label="Path"
          tooltip="The interface path to query"
        />
      </div>
    );
  }
}
